package models

import "time"

type Usuario struct {
	ID            uint `gorm:"primaryKey"`
	RolID         uint
	NombreUsuario string    `gorm:"size:100;not null"`
	Email         string    `gorm:"size:150;unique;not null"`
	Password      string    `gorm:"size:255;not null"`
	FechaRegistro time.Time `gorm:"autoCreateTime"`

	Rol Rol `gorm:"foreignKey:RolID"`
}
