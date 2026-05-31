// Punto de entrada principal del servidor backend.
// Inicializa la base de datos, configura CORS, registra todas las rutas
// y levanta el servidor HTTP con Gin.
package main

import (
	"ProyectoGinBack/config"      // Configuración de BD y migraciones
	"ProyectoGinBack/controllers" // Handlers HTTP de cada recurso
	"ProyectoGinBack/middlewares" // Middlewares de autenticación y roles
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors" // Middleware CORS para permitir peticiones del frontend
	"github.com/gin-gonic/gin"    // Framework HTTP principal
	"github.com/joho/godotenv"    // Carga de variables de entorno desde archivo .env
)

func main() {

	// -------------------------------------------------------------------------
	// 1. CARGA DE VARIABLES DE ENTORNO
	// Debe ejecutarse antes de cualquier otra inicialización, ya que
	// config.ConectarDB() y otros módulos dependen de variables como
	// DATABASE_URL, JWT_SECRET, PORT, etc.
	// -------------------------------------------------------------------------
	err := godotenv.Load()
	if err != nil {
		// No es un error fatal: en producción (Railway, Render, etc.)
		// las variables se inyectan directamente en el entorno del proceso.
		log.Println("⚠️ Advertencia: No se pudo cargar el archivo .env (revisa si existe en la raíz)")
	}

	// -------------------------------------------------------------------------
	// 2. INICIALIZACIÓN DE BASE DE DATOS
	// ConectarDB abre la conexión con el ORM (GORM + PostgreSQL/SQLite).
	// MigrarModelos ejecuta las migraciones automáticas: crea o actualiza
	// las tablas según las structs definidas en /models.
	// -------------------------------------------------------------------------
	config.ConectarDB()
	config.MigrarModelos()

	// -------------------------------------------------------------------------
	// 3. INSTANCIA DEL ROUTER GIN
	// gin.Default() incluye por defecto los middlewares de Logger y Recovery.
	// - Logger: imprime cada request en consola.
	// - Recovery: captura panics y devuelve HTTP 500 en lugar de crashear.
	// -------------------------------------------------------------------------
	r := gin.Default()

	// -------------------------------------------------------------------------
	// 4. CONFIGURACIÓN DE CORS
	// Restringe qué orígenes, métodos y cabeceras pueden acceder a la API.
	// Orígenes permitidos:
	//   - http://localhost:5173  → frontend React+Vite en desarrollo local
	//   - https://tuhiro33.github.io → frontend desplegado en GitHub Pages
	// MaxAge: el navegador cachea los resultados del preflight OPTIONS por 12h,
	// reduciendo llamadas extra al servidor.
	// -------------------------------------------------------------------------
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "https://tuhiro33.github.io"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true, // Permite enviar cookies y cabeceras Authorization
		MaxAge:           12 * time.Hour,
	}))

	// -------------------------------------------------------------------------
	// 5. RUTA DE HEALTH CHECK
	// Endpoint público para verificar que el servidor está activo.
	// Útil para monitoreo, balanceadores de carga y CI/CD pipelines.
	// -------------------------------------------------------------------------
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// =========================================================================
	// RUTAS DE LA API
	// Patrón general:
	//   - Rutas públicas: accesibles sin token (consultas de lectura, login, registro)
	//   - Rutas protegidas (grupo 'auth'): requieren JWT válido en header Authorization
	//   - Rutas admin: requieren JWT válido + rol con ID 2 (administrador)
	// =========================================================================

	// -------------------------------------------------------------------------
	// PUBLICACIONES
	// Recurso principal: cartas TCG (Magic/Pokémon) publicadas para venta/intercambio.
	// GET listado es público; crear, editar y eliminar requieren autenticación.
	// -------------------------------------------------------------------------
	r.GET("/publicaciones", controllers.ObtenerPublicaciones)        // Listado público de todas las publicaciones
	r.GET("/publicaciones/:id", controllers.ObtenerPublicacionPorID) // Detalle público de una publicación por ID

	// Grupo protegido: todas las rutas dentro requieren token JWT válido.
	// El middleware AuthMiddleware() valida la firma y expiración del token,
	// e inyecta los datos del usuario autenticado en el contexto de Gin.
	auth := r.Group("/")
	auth.Use(middlewares.AuthMiddleware())
	{
		auth.POST("/publicaciones", controllers.CrearPublicacion)           // Crear nueva publicación de carta
		auth.PUT("/publicaciones/:id", controllers.ActualizarPublicacion)   // Editar publicación propia
		auth.DELETE("/publicaciones/:id", controllers.EliminarPublicacion)  // Eliminar publicación propia
		auth.GET("/mis-publicaciones", controllers.ObtenerMisPublicaciones) // Listar publicaciones del usuario autenticado
	}

	// -------------------------------------------------------------------------
	// USUARIOS
	// -------------------------------------------------------------------------
	r.POST("/usuarios", controllers.CrearUsuario) // Registro directo de usuario (sin token)
	r.POST("/login", controllers.Login)           // Autenticación: devuelve JWT si las credenciales son válidas
	r.POST("/register", controllers.Register)     // Registro con validaciones adicionales

	// Rutas públicas de perfil (solo lectura, sin datos sensibles)
	r.GET("/usuarios/coleccionistas", controllers.ObtenerUsuariosConColeccion) // Usuarios que tienen colección registrada
	r.GET("/usuarios/perfil/:usuarioId", controllers.ObtenerPerfilPublico)     // Perfil público de un usuario específico

	auth.GET("/me", controllers.GetProfile)               // Perfil completo del usuario autenticado (datos propios)
	auth.GET("/usuarios", controllers.ObtenerUsuarios)    // Listado de usuarios (requiere autenticación)
	auth.PUT("/usuarios", controllers.ActualizarUsuario)  // Actualizar datos del usuario autenticado
	auth.DELETE("/usuarios", controllers.EliminarUsuario) // Eliminar cuenta del usuario autenticado

	// -------------------------------------------------------------------------
	// COLECCIÓN DE CARTAS
	// Permite a cada usuario gestionar su biblioteca personal de cartas TCG.
	// SincronizarCarta: obtiene datos de la carta desde una API externa (Scryfall / PokéAPI)
	// y la guarda localmente en caché para no repetir llamadas externas.
	// -------------------------------------------------------------------------
	auth.POST("/cartas/sincronizar", controllers.SincronizarCarta)         // Sincroniza datos de carta desde API externa al caché local
	auth.GET("/coleccion/:usuarioId", controllers.ObtenerColeccionUsuario) // Obtiene la colección de un usuario por su ID
	auth.POST("/coleccion", controllers.AgregarAColeccion)                 // Agrega una carta a la colección del usuario autenticado
	auth.DELETE("/coleccion/:id", controllers.EliminarDeColeccion)         // Elimina una carta de la colección por ID de entrada

	// -------------------------------------------------------------------------
	// NOTIFICACIONES DE INTERCAMBIO
	// Subgrupo /api dentro del grupo auth para aislar este módulo.
	// Envía una notificación al usuario destino cuando se propone un intercambio de cartas.
	// -------------------------------------------------------------------------
	api := auth.Group("/api")
	{
		api.POST("/intercambio/notificar", controllers.NotificarIntercambio) // Notifica a un usuario sobre una propuesta de intercambio
	}

	// -------------------------------------------------------------------------
	// TRANSACCIONES
	// Registro del historial de compras/ventas entre usuarios.
	// -------------------------------------------------------------------------
	auth.POST("/transacciones", controllers.CrearTransaccion)                            // Registra una nueva transacción de compra/venta
	auth.GET("/transacciones/historial/:usuarioId", controllers.ObtenerHistorialCompras) // Historial de compras de un usuario específico

	// Marcar publicación como vendida: aplica AuthMiddleware directamente
	// (fuera del grupo auth porque tiene su propio middleware inline)
	r.PUT("/publicaciones/:id/vendida", middlewares.AuthMiddleware(), controllers.MarcarComoVendida)

	// -------------------------------------------------------------------------
	// ROLES (lectura pública)
	// -------------------------------------------------------------------------
	r.GET("/roles", controllers.ObtenerRoles) // Lista de roles disponibles en el sistema (público)

	// -------------------------------------------------------------------------
	// UPLOAD DE IMÁGENES
	// Permite subir imágenes de cartas (portadas de publicaciones).
	// TODO: considerar mover dentro del grupo 'auth' para evitar uploads anónimos.
	// -------------------------------------------------------------------------
	r.POST("/upload", controllers.UploadImage)

	// -------------------------------------------------------------------------
	// RUTAS DE ADMINISTRACIÓN
	// Grupo doblemente protegido:
	//   1. AuthMiddleware: valida que el token JWT sea válido
	//   2. RequireRoles(2): verifica que el usuario tenga rol de administrador (ID=2)
	// Solo los administradores pueden crear, editar o eliminar roles del sistema.
	// -------------------------------------------------------------------------
	admin := r.Group("/admin")
	admin.Use(middlewares.AuthMiddleware())
	admin.Use(middlewares.RequireRoles(2)) // ID 2 = rol Administrador
	{
		admin.POST("/roles", controllers.CrearRol)          // Crear nuevo rol
		admin.PUT("/roles/:id", controllers.ActualizarRol)  // Editar nombre/permisos de un rol existente
		admin.DELETE("/roles/:id", controllers.EliminarRol) // Eliminar un rol del sistema
	}

	// -------------------------------------------------------------------------
	// INICIO DEL SERVIDOR
	// Toma el puerto de la variable de entorno PORT (definida por plataformas
	// como Railway o Render). Si no está definida, usa 8080 por defecto.
	// -------------------------------------------------------------------------
	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "8080" // Puerto por defecto para desarrollo local
	}

	log.Printf("🚀 Servidor corriendo en el puerto %s", puerto)
	r.Run(":" + puerto)
}
