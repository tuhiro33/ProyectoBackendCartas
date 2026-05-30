package main

import (
	"ProyectoGinBack/config"
	"ProyectoGinBack/controllers"
	"ProyectoGinBack/middlewares"
	"log" // 1. Agrega este import nativo si no está

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv" // 2. IMPORTANTE: Agrega la librería godotenv aquí
)

func main() {
	// 3. ¡CRÍTICO! Cargar las variables del .env antes de inicializar la DB y las rutas
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Advertencia: No se pudo cargar el archivo .env (revisa si existe en la raíz)")
	}

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

	auth := r.Group("/")
	auth.Use(middlewares.AuthMiddleware()) // Grupo protegido por token JWT

	auth.POST("/publicaciones", controllers.CrearPublicacion)
	auth.PUT("/publicaciones/:id", controllers.ActualizarPublicacion)
	auth.DELETE("/publicaciones/:id", controllers.EliminarPublicacion)
	auth.GET("/mis-publicaciones", controllers.ObtenerMisPublicaciones)
	r.GET("/publicaciones/:id", controllers.ObtenerPublicacionPorID)

	//RUTAS DE USUARIO ----------------------------------------------------------------
	r.POST("/usuarios", controllers.CrearUsuario)
	auth.GET("/me", controllers.GetProfile)
	auth.GET("/usuarios", controllers.ObtenerUsuarios)
	auth.PUT("/usuarios", controllers.ActualizarUsuario)
	auth.DELETE("/usuarios", controllers.EliminarUsuario)

	// Colección
	auth.POST("/cartas/sincronizar", controllers.SincronizarCarta)
	auth.GET("/coleccion/:usuarioId", controllers.ObtenerColeccionUsuario)
	auth.POST("/coleccion", controllers.AgregarAColeccion)
	auth.DELETE("/coleccion/:id", controllers.EliminarDeColeccion)

	// NOTIFICACIONES DE INTERCAMBIO (Metidas dentro del grupo 'auth' para asegurar el Token)
	api := auth.Group("/api")
	{
		api.POST("/intercambio/notificar", controllers.NotificarIntercambio)
	}

	// Transacciones
	auth.POST("/transacciones", controllers.CrearTransaccion)
	auth.GET("/transacciones/historial/:usuarioId", controllers.ObtenerHistorialCompras)
	r.PUT("/publicaciones/:id/vendida", middlewares.AuthMiddleware(), controllers.MarcarComoVendida)

	//Login
	r.POST("/login", controllers.Login)
	//Register
	r.POST("/register", controllers.Register)

	// rutas públicas
	r.GET("/roles", controllers.ObtenerRoles)
	r.GET("/usuarios/coleccionistas", controllers.ObtenerUsuariosConColeccion)
	r.GET("/usuarios/perfil/:usuarioId", controllers.ObtenerPerfilPublico)

	// rutas admin (protegidas)
	admin := r.Group("/admin")
	admin.Use(middlewares.AuthMiddleware())
	admin.Use(middlewares.RequireRoles(2))
	{
		admin.POST("/roles", controllers.CrearRol)
		admin.PUT("/roles/:id", controllers.ActualizarRol)
		admin.DELETE("/roles/:id", controllers.EliminarRol)
	}

	r.POST("/upload", controllers.UploadImage)

	r.Run(":8080")
}
