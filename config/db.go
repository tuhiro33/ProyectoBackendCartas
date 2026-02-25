package config

import (
	"log"

	"ProyectoGinBack/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConectarDB() {
	dsn := "host=localhost user=postgres password=Nada123@ dbname=proyectocartones port=5432 sslmode=disable"

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Error conectando a PostgreSQL:", err)
	}

	log.Println("Conectado a PostgreSQL")
}

func MigrarModelos() {
	DB.AutoMigrate(
		&models.Rol{},
		&models.Usuario{},
		&models.CartaCache{},
		&models.ColeccionUsuario{},
		&models.PublicacionVenta{},
		&models.Transaccion{},
	)
}
