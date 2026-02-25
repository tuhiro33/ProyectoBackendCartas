package main

import (
	"ProyectoGinBack/config"
	"ProyectoGinBack/controllers"

	"github.com/gin-gonic/gin"
)

func main() {
	config.ConectarDB()
	config.MigrarModelos()

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	//RUTAS
	r.GET("/publicaciones", controllers.ObtenerPublicaciones)
	r.POST("/publicaciones", controllers.CrearPublicacion)
	r.GET("/publicaciones/:id", controllers.ObtenerPublicacionPorID)

	r.Run(":8080")
}
