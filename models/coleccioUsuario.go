package models

type ColeccionUsuario struct {
	ID         uint `gorm:"primaryKey"`
	UsuarioID  uint
	CartaApiID string `gorm:"size:100"`
	Cantidad   int
	EsFoil     bool

	Usuario Usuario    `gorm:"foreignKey:UsuarioID"`
	Carta   CartaCache `gorm:"foreignKey:CartaApiID;references:ApiID"`
}
