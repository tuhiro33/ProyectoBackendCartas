package main

import (
	"ProyectoGinBack/config"
	"ProyectoGinBack/controllers"
	"ProyectoGinBack/middlewares"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.ConectarDB()
	config.MigrarModelos()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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
	//r.GET("/usuarios", controllers.ObtenerUsuarios)
	auth.GET("/usuarios", controllers.ObtenerUsuarios)
	auth.PUT("/usuarios", controllers.ActualizarUsuario)
	auth.DELETE("/usuarios", controllers.EliminarUsuario)

	//LOgin
	r.POST("/login", controllers.Login)
	//Register
	r.POST("/register", controllers.Register)

	// rutas públicas
	r.GET("/roles", controllers.ObtenerRoles)

	// rutas admin (protegidas)
	admin := r.Group("/admin")
	admin.Use(middlewares.AuthMiddleware()) // ya valida token
	admin.Use(middlewares.RequireRoles(2))  // solo rol 2 = admin (ajusta el ID si es otro)
	{
		admin.POST("/roles", controllers.CrearRol)
		admin.PUT("/roles/:id", controllers.ActualizarRol)
		admin.DELETE("/roles/:id", controllers.EliminarRol)
	}

	r.Run(":8080")
}
