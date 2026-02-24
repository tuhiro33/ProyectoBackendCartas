package config

import (
	"ProyectoGinBack/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConectarDB() {
	dsn := "host=localhost user=postgres password=TU_PASSWORD dbname=mi_proyecto_db port=5432 sslmode=disable"
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		panic("Fallo al conectar a la DB")
	}

	database.AutoMigrate(&models.Usuario{})
	DB = database
}
