package models

type CartaCache struct {
	ApiID  string `gorm:"primaryKey;size:100"`
	Juego  string `gorm:"size:100"`
	Nombre string `gorm:"size:150"`
}
