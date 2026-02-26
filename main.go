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

	//RUTAS DE PUBLICACION -----------------------------------------------------------
	r.GET("/publicaciones", controllers.ObtenerPublicaciones)

	auth := r.Group("/") //aca
	auth.Use(middlewares.AuthMiddleware())
	auth.POST("/publicaciones", controllers.CrearPublicacion) //aqui
	auth.PUT("/publicaciones/:id", controllers.ActualizarPublicacion)
	auth.DELETE("/publicaciones/:id", controllers.EliminarPublicacion)
	auth.GET("/mis-publicaciones", controllers.ObtenerMisPublicaciones)
	//r.POST("/publicaciones", controllers.CrearPublicacion)    // este borrar
	r.GET("/publicaciones/:id", controllers.ObtenerPublicacionPorID)

	//RUTAS DE USUARIO ----------------------------------------------------------------
	r.POST("/usuarios", controllers.CrearUsuario)
	r.GET("/usuarios", controllers.ObtenerUsuarios)
	auth.GET("/usuarios", controllers.ObtenerUsuarios)
	auth.PUT("/usuarios", controllers.ActualizarUsuario)
	auth.DELETE("/usuarios", controllers.EliminarUsuario)

	//LOgin
	r.POST("/login", controllers.Login)

	r.Run(":8080")
}
