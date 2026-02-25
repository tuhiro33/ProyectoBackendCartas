package models

import "time"

type PublicacionVenta struct {
	ID                uint `gorm:"primaryKey"`
	VendedorID        uint
	ColeccionID       uint
	Precio            float64   `gorm:"type:decimal(10,2)"`
	EstadoCarta       string    `gorm:"size:50"`
	FotoURL           string    `gorm:"size:512" json:"foto_url"`
	EstadoPublicacion string    `gorm:"size:50"`
	FechaPublicacion  time.Time `gorm:"autoCreateTime"`

	Vendedor  Usuario          `gorm:"foreignKey:VendedorID"`
	Coleccion ColeccionUsuario `gorm:"foreignKey:ColeccionID"`
}
