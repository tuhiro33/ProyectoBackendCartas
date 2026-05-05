package models

type ColeccionUsuario struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	UsuarioID  uint       `json:"usuario_id"`
	CartaApiID string     `json:"carta_api_id" gorm:"size:100"`
	Cantidad   int        `json:"cantidad"`
	EsFoil     bool       `json:"es_foil"`
	Carta      CartaCache `json:"carta" gorm:"foreignKey:CartaApiID;references:ApiID"`
}
