package main

import (
	"ProyectoGinBack/config"
	"ProyectoGinBack/controllers"
	"ProyectoGinBack/middlewares"

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

	auth := r.Group("/") //aca
	auth.Use(middlewares.AuthMiddleware())
	auth.POST("/publicaciones", controllers.CrearPublicacion) //aqui
	//r.POST("/publicaciones", controllers.CrearPublicacion)    // este borrar
	r.GET("/publicaciones/:id", controllers.ObtenerPublicacionPorID)

	r.POST("/usuarios", controllers.CrearUsuario)
	r.GET("/usuarios", controllers.ObtenerUsuarios)
	r.POST("/login", controllers.Login)

	r.Run(":8080")
}
