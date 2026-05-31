// Controlador de intercambio: gestiona las notificaciones por email cuando
// un usuario hace una oferta por la carta publicada de otro usuario.
//
// Usa Resend (servicio externo de envío de emails) como proveedor.
// El envío se hace de forma asíncrona con una goroutine para no bloquear
// la respuesta HTTP mientras se espera la llamada a la API de Resend.
//
// Rutas que maneja (ver main.go):
//
//	POST /api/intercambio/notificar → NotificarIntercambio (protegida con JWT)
package controllers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/resend/resend-go/v3"
)

// IntercambioRequest define la estructura del JSON que llega desde el frontend
// cuando un usuario quiere notificar su interés en una carta.
//
// Las etiquetas `json:"..."` indican cómo se llama cada campo en el JSON entrante
// (camelCase desde TypeScript) y cómo Go los mapea a sus campos internos.
// `binding:"required"` le dice a ShouldBindJSON que rechace la petición
// con HTTP 400 si ese campo viene vacío o ausente en el JSON.
type IntercambioRequest struct {
	NombreCarta        string  `json:"nombreCarta"        binding:"required"`
	Precio             float64 `json:"precio"             binding:"required"`
	EstadoCarta        string  `json:"estadoCarta"        binding:"required"`
	NombreDestinatario string  `json:"nombreDestinatario" binding:"required"`

	// CorreoComprador es el email del usuario interesado en la carta.
	// Se incluye en el cuerpo del email para que el vendedor pueda contactarlo.
	// ⚠️  No valida que sea un email con formato válido (binding:"required,email")
	// solo verifica que no esté vacío. Un valor como "noesuncorreo" pasaría la validación.
	CorreoComprador string `json:"correoComprador" binding:"required"`
}

// NotificarIntercambio es el handler HTTP del endpoint POST /api/intercambio/notificar.
//
// Flujo:
//  1. Deserializa y valida el JSON del body con ShouldBindJSON.
//     ShouldBindJSON lee c.Request.Body, intenta mapear los campos JSON
//     a la struct IntercambioRequest y verifica los binding:"required".
//     Si algo falla devuelve error y el handler responde con 400.
//  2. Lanza el envío del email en segundo plano con una goroutine (go ...).
//     Una goroutine es una función que Go ejecuta de forma concurrente —
//     el handler responde al cliente inmediatamente sin esperar que el email
//     se envíe, lo que evita que el usuario espere 1-2 segundos por la API de Resend.
//  3. Responde con 200 OK al cliente indicando que la oferta fue recibida.
//
// ⚠️  LIMITACIÓN DE LA GOROUTINE ASÍNCRONA:
// Como el email se envía en segundo plano, el cliente recibe 200 OK sin saber
// si el envío realmente funcionó. Si Resend falla (API key inválida, servicio caído),
// el error solo queda en los logs del servidor y el usuario cree que todo salió bien.
// Para un sistema de producción considerar:
//   - Retornar error si el envío falla (respuesta síncrona)
//   - O usar una cola de tareas para reintentos automáticos
func NotificarIntercambio(c *gin.Context) {
	var req IntercambioRequest

	// ShouldBindJSON deserializa el body JSON en req.
	// El & (ampersand) pasa la dirección de memoria de req para que
	// ShouldBindJSON pueda escribir directamente en sus campos.
	// Sin el & se pasaría una copia y los cambios se perderían.
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de la carta inválidos o incompletos"})
		return
	}

	// La palabra clave 'go' lanza enviarNotificacionResend en una goroutine separada.
	// El código continúa a la siguiente línea sin esperar que la función termine.
	// req se pasa por valor (copia) para evitar condiciones de carrera — la goroutine
	// tiene su propia copia de los datos y no depende de que req siga en memoria.
	go enviarNotificacionResend(req)

	// Responde inmediatamente al cliente mientras el email se procesa en paralelo
	c.JSON(http.StatusOK, gin.H{"message": "Oferta enviada y correo en proceso"})
}

// enviarNotificacionResend es una función privada (nombre en minúscula) que
// construye y envía el email de notificación usando la API de Resend.
//
// En Go, las funciones que empiezan con minúscula son privadas al paquete —
// solo pueden llamarse desde este mismo archivo o desde otros archivos del
// paquete controllers. No son accesibles desde otros paquetes.
//
// Se ejecuta siempre desde una goroutine (llamada con 'go' en NotificarIntercambio)
// por lo que no tiene acceso al contexto de Gin (c *gin.Context) ni puede
// modificar la respuesta HTTP — solo puede loguear errores en el servidor.
//
// ⚠️  El correo destinatario está hardcodeado como "compucare.contacto@gmail.com".
// En producción debería usar datos.CorreoDestinatario (el email del vendedor
// dueño de la carta), que actualmente no está en IntercambioRequest.
// El sistema notifica siempre al mismo correo fijo, no al vendedor real.
func enviarNotificacionResend(datos IntercambioRequest) {
	// Lee la API key de Resend desde variables de entorno.
	// Si RESEND_API_KEY no está definida, apiKey será "" y el cliente
	// de Resend fallará al intentar autenticarse — el error se captura abajo.
	apiKey := os.Getenv("RESEND_API_KEY")
	client := resend.NewClient(apiKey)

	// Construye el cuerpo HTML del email usando fmt.Sprintf.
	// Cada %s se reemplaza por un string y %0.2f por el precio con 2 decimales.
	// El HTML está inline (sin archivo de plantilla externo) lo cual es simple
	// pero dificulta cambiar el diseño sin recompilar el servidor.
	htmlBody := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 500px; border: 1px solid #e2e8f0; border-radius: 8px; padding: 20px;">
			<h2 style="color: #1e3a8a;">¡Hola, %s!</h2>
			<p>Tienes una nueva oferta por tu carta:</p>
			<hr style="border: 0; border-top: 1px solid #e2e8f0; margin: 15px 0;">
			<p><strong>🃏 Carta:</strong> %s</p>
			<p><strong>✨ Estado:</strong> %s</p>
			<p><strong>💰 Precio:</strong> $%0.2f</p>
			<hr style="border: 0; border-top: 1px solid #e2e8f0; margin: 15px 0;">
			<p><strong>📩 Interesado:</strong> %s</p>
		</div>
	`,
		datos.NombreDestinatario, // %s → nombre del vendedor que recibe el email
		datos.NombreCarta,        // %s → nombre de la carta ofertada
		datos.EstadoCarta,        // %s → estado físico de la carta (ej: "Mint", "Bueno")
		datos.Precio,             // %0.2f → precio ofrecido, con 2 decimales (ej: 15.00)
		datos.CorreoComprador,    // %s → email del comprador interesado
	)

	// SendEmailRequest es la struct de la librería resend-go que define
	// los parámetros del email a enviar.
	// From: debe ser un dominio verificado en Resend. "onboarding@resend.dev"
	// es el dominio de pruebas de Resend — solo funciona en modo sandbox.
	// En producción debe cambiarse a un dominio propio verificado.
	params := &resend.SendEmailRequest{
		From:    "onboarding@resend.dev",
		To:      []string{"compucare.contacto@gmail.com"}, // ⚠️  Hardcodeado — ver advertencia arriba
		Subject: fmt.Sprintf("¡Nueva oferta por tu %s!", datos.NombreCarta),
		Html:    htmlBody,
	}

	// client.Emails.Send hace la llamada HTTP a la API de Resend.
	// El primer valor de retorno (ignorado con _) sería la respuesta con el ID del email.
	// Si hay error se imprime en los logs del servidor pero no llega al cliente
	// porque esta función corre en una goroutine desconectada del ciclo HTTP.
	_, err := client.Emails.Send(params)
	if err != nil {
		// fmt.Printf escribe en stdout del servidor — visible en los logs de Railway/Render
		// pero invisible para el usuario que ya recibió su 200 OK.
		fmt.Printf("Error enviando email: %v\n", err)
	}
}
