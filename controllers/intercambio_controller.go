package controllers

import (
	"fmt"
	"net/http"

	"os"

	"github.com/gin-gonic/gin"
	"github.com/resend/resend-go/v3"
)

// IntercambioRequest define los datos que vienen desde TypeScript
type IntercambioRequest struct {
	NombreCarta        string  `json:"nombreCarta" binding:"required"`
	Precio             float64 `json:"precio" binding:"required"`
	EstadoCarta        string  `json:"estadoCarta" binding:"required"`
	NombreDestinatario string  `json:"nombreDestinatario" binding:"required"`
	CorreoComprador    string  `json:"correoComprador" binding:"required"`
}

// NotificarIntercambio Handler para el POST de la API
func NotificarIntercambio(c *gin.Context) {
	var req IntercambioRequest

	// Validar el JSON entrante
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de la carta inválidos o incompletos"})
		return
	}

	// Ejecutar en segundo plano con una Goroutine
	go enviarNotificacionResend(req)

	c.JSON(http.StatusOK, gin.H{"message": "Oferta enviada y correo en proceso"})
}

// Función interna (privada, empieza con minúscula) para procesar el email
func enviarNotificacionResend(datos IntercambioRequest) {
	apiKey := os.Getenv("RESEND_API_KEY")
	client := resend.NewClient(apiKey)
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
	`, datos.NombreDestinatario, datos.NombreCarta, datos.EstadoCarta, datos.Precio, datos.CorreoComprador)

	params := &resend.SendEmailRequest{
		From:    "onboarding@resend.dev",
		To:      []string{"compucare.contacto@gmail.com"}, // Tu correo de pruebas de Resend
		Subject: fmt.Sprintf("¡Nueva oferta por tu %s!", datos.NombreCarta),
		Html:    htmlBody,
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		fmt.Printf("Error enviando email: %v\n", err)
	}
}
