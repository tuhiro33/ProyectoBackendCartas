// DTO de publicaciones: define la estructura de respuesta del marketplace
// y la función que convierte el modelo interno a esa estructura.
//
// Este es el DTO más complejo del proyecto porque aplana dos niveles
// de relaciones anidadas:
//
//	models.PublicacionVenta
//	  → Vendedor (Usuario)
//	  → Coleccion (ColeccionUsuario)
//	      → Carta (CartaCache)
//
// En lugar de devolver esa jerarquía completa, MapPublicacionToDTO
// la aplana en un objeto plano con solo los campos necesarios para el frontend.
package dto

import (
	"ProyectoGinBack/models"
	"time"
)

// PublicacionResponse es la representación pública de una publicación
// en el marketplace. Se usa en respuestas de:
//
//	GET /publicaciones
//	GET /publicaciones/:id
//	GET /mis-publicaciones
//
// Aplana tres modelos relacionados en una sola struct para que el frontend
// no tenga que navegar objetos anidados profundos.
//
// Debe coincidir con las interfaces del frontend:
//
//	PublicacionVenta en ventasService.ts
//	PublicacionVenta en PerfilPublico.tsx
type PublicacionResponse struct {
	ID                uint         `json:"id"`
	Precio            float64      `json:"precio"`
	EstadoCarta       string       `json:"estado_carta"`
	FotoURL           string       `json:"foto_url"`
	EstadoPublicacion string       `json:"estado_publicacion"`
	FechaPublicacion  time.Time    `json:"fecha_publicacion"`
	Vendedor          VendedorDTO  `json:"vendedor"`  // Datos básicos del vendedor
	Coleccion         ColeccionDTO `json:"coleccion"` // Datos de la carta publicada
}

// VendedorDTO expone solo los datos necesarios del vendedor
// para mostrar en cada tarjeta del marketplace.
// Excluye Email, Password, FechaRegistro y otros campos sensibles de Usuario.
//
// ⚠️  El campo json:"nombre" devuelve NombreUsuario del modelo —
// el frontend en ventasService.ts espera "nombre" (no "nombre_usuario").
// MapPublicacionToDTO hace esta conversión explícitamente.
type VendedorDTO struct {
	ID     uint   `json:"id"`
	Nombre string `json:"nombre"` // Mapeado desde models.Usuario.NombreUsuario
}

// ColeccionDTO aplana ColeccionUsuario + CartaCache en una sola struct.
// Los campos carta_nombre, carta_juego y carta_imagen vienen de CartaCache
// (dos niveles de profundidad) pero se exponen al mismo nivel que id y cantidad.
//
// Esto requiere que el controlador haga Preload("Coleccion.Carta") —
// sin ese Preload, CartaNombre, CartaJuego y CartaImagen serán cadenas vacías.
//
// ⚠️  CartaJuego debería ser 'magic' | 'pokemon' para consistencia con
// las interfaces TypeScript del frontend, pero en Go se tipea como string
// genérico. Si se pasa un valor distinto desde la BD, el frontend
// no lo reconocerá en sus comparaciones (carta_juego === 'magic').
type ColeccionDTO struct {
	ID          uint   `json:"id"`
	Cantidad    uint   `json:"cantidad"`
	CartaNombre string `json:"carta_nombre"` // Viene de Coleccion.Carta.Nombre
	CartaJuego  string `json:"carta_juego"`  // Viene de Coleccion.Carta.Juego
	CartaImagen string `json:"carta_imagen"` // Viene de Coleccion.Carta.UrlImagen
}

// MapPublicacionToDTO convierte models.PublicacionVenta al DTO de respuesta.
//
// Precondiciones — el modelo debe tener precargadas TODAS estas relaciones
// antes de llamar esta función, de lo contrario los campos quedarán vacíos:
//
//	config.DB.
//	  Preload("Vendedor").         // para Vendedor.ID y Vendedor.NombreUsuario
//	  Preload("Coleccion").        // para Coleccion.ID y Coleccion.Cantidad
//	  Preload("Coleccion.Carta").  // para Carta.Nombre, Carta.Juego, Carta.UrlImagen
//	  Find(&publicaciones)
//
// Si algún Preload falta, el campo correspondiente será el zero value de Go:
//   - string  → ""   (cadena vacía en JSON)
//   - uint    → 0
//   - float64 → 0.0
//
// El frontend recibirá los datos sin error pero con valores incorrectos,
// lo que puede ser difícil de depurar.
func MapPublicacionToDTO(p models.PublicacionVenta) PublicacionResponse {
	return PublicacionResponse{
		ID:                p.ID,
		Precio:            p.Precio,
		EstadoCarta:       p.EstadoCarta,
		FotoURL:           p.FotoURL,
		EstadoPublicacion: p.EstadoPublicacion,
		FechaPublicacion:  p.FechaPublicacion,

		// VendedorDTO aplana Usuario a solo ID y Nombre.
		// p.Vendedor.NombreUsuario → json:"nombre" (no "nombre_usuario")
		// para coincidir con la interfaz VendedorDTO del frontend.
		Vendedor: VendedorDTO{
			ID:     p.Vendedor.ID,
			Nombre: p.Vendedor.NombreUsuario, // Renombrado intencionalmente
		},

		// ColeccionDTO aplana ColeccionUsuario + CartaCache.
		// p.Coleccion.Carta.* requiere Preload("Coleccion.Carta") previo —
		// si no se hizo el Preload, estos tres campos serán cadenas vacías.
		// uint(p.Coleccion.Cantidad) convierte int a uint porque
		// ColeccionUsuario.Cantidad es int pero ColeccionDTO.Cantidad es uint.
		// ⚠️  Si Cantidad fuera negativo (no debería ocurrir pero no hay
		// validación en el modelo), la conversión a uint daría un número enorme.
		Coleccion: ColeccionDTO{
			ID:          p.Coleccion.ID,
			Cantidad:    uint(p.Coleccion.Cantidad),  // int → uint
			CartaNombre: p.Coleccion.Carta.Nombre,    // Requiere Preload("Coleccion.Carta")
			CartaJuego:  p.Coleccion.Carta.Juego,     // Requiere Preload("Coleccion.Carta")
			CartaImagen: p.Coleccion.Carta.UrlImagen, // Requiere Preload("Coleccion.Carta")
		},
	}
}
