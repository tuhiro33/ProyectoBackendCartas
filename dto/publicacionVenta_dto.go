package dto

import (
	"ProyectoGinBack/models"
	"time"
)

type PublicacionResponse struct {
	ID                uint         `json:"id"`
	Precio            float64      `json:"precio"`
	EstadoCarta       string       `json:"estado_carta"`
	FotoURL           string       `json:"foto_url"`
	EstadoPublicacion string       `json:"estado_publicacion"`
	FechaPublicacion  time.Time    `json:"fecha_publicacion"`
	Vendedor          VendedorDTO  `json:"vendedor"`
	Coleccion         ColeccionDTO `json:"coleccion"`
}

type VendedorDTO struct {
	ID     uint   `json:"id"`
	Nombre string `json:"nombre"`
}

type ColeccionDTO struct {
	ID          uint   `json:"id"`
	Cantidad    uint   `json:"cantidad"`
	CartaNombre string `json:"carta_nombre"`
	CartaJuego  string `json:"carta_juego"`
	CartaImagen string `json:"carta_imagen"`
}

func MapPublicacionToDTO(p models.PublicacionVenta) PublicacionResponse {
	return PublicacionResponse{
		ID:                p.ID,
		Precio:            p.Precio,
		EstadoCarta:       p.EstadoCarta,
		FotoURL:           p.FotoURL,
		EstadoPublicacion: p.EstadoPublicacion,
		FechaPublicacion:  p.FechaPublicacion,
		Vendedor: VendedorDTO{
			ID:     p.Vendedor.ID,
			Nombre: p.Vendedor.NombreUsuario,
		},
		Coleccion: ColeccionDTO{
			ID:          p.Coleccion.ID,
			Cantidad:    uint(p.Coleccion.Cantidad),
			CartaNombre: p.Coleccion.Carta.Nombre, // ← Preload necesario
			CartaJuego:  p.Coleccion.Carta.Juego,
			CartaImagen: p.Coleccion.Carta.UrlImagen,
		},
	}
}
