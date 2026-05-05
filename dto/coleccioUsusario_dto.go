package dto

type AgregarCartaRequest struct {
	UsuarioID uint `json:"usuario_id" binding:"required"`
	Cantidad  int  `json:"cantidad" binding:"required"`
	EsFoil    bool `json:"es_foil"`

	// Anidamos los detalles de la carta para asegurar la integridad en la DB
	Carta struct {
		ApiID     string `json:"api_id" binding:"required"`
		Juego     string `json:"juego" binding:"required"`
		Nombre    string `json:"nombre" binding:"required"`
		UrlImagen string `json:"url_imagen" binding:"required"`
	} `json:"carta" binding:"required"`
}
