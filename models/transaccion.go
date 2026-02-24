package models

import "time"

type Transaccion struct {
	ID            uint `gorm:"primaryKey"`
	PublicacionID uint
	CompradorID   uint
	PrecioFinal   float64   `gorm:"type:decimal(10,2)"`
	FechaCompra   time.Time `gorm:"autoCreateTime"`
	EstadoPago    string    `gorm:"size:50"`

	Publicacion PublicacionVenta `gorm:"foreignKey:PublicacionID"`
	Comprador   Usuario          `gorm:"foreignKey:CompradorID"`
}
