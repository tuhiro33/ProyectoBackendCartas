package models

import "gorm.io/gorm"

type Usuario struct {
	gorm.Model
	Nombre string `json:"nombre"`
	Email  string `json:"email" gorm:"unique"`
}
