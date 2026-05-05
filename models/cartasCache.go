package models

type CartaCache struct {
	ApiID     string `json:"api_id" gorm:"primaryKey;size:100"`
	Juego     string `json:"juego" gorm:"size:100"`
	Nombre    string `json:"nombre" gorm:"size:150"`
	UrlImagen string `json:"url_imagen" gorm:"size:255"` // ← añadir este campo
}
