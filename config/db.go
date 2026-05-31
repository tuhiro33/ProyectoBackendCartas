// Paquete config: gestiona la conexión a la base de datos PostgreSQL
// mediante el ORM GORM y ejecuta las migraciones automáticas de los modelos.
package config

import (
	"log"
	"os"

	"ProyectoGinBack/models" // Structs que representan las tablas de la BD

	"gorm.io/driver/postgres" // Driver PostgreSQL para GORM
	"gorm.io/gorm"            // ORM principal
)

// DB es la instancia global de GORM.
// Al ser una variable de paquete, todos los controladores pueden importarla
// con config.DB y ejecutar consultas directamente sobre ella.
var DB *gorm.DB

// ConectarDB inicializa la conexión a PostgreSQL usando GORM.
//
// Estrategia de conexión (por prioridad):
//  1. Lee la variable de entorno DATABASE_URL (definida por Railway/Render en producción).
//  2. Si la variable está vacía (entorno local / .env no cargado), usa una DSN
//     hardcodeada apuntando a PostgreSQL local.
//
// ⚠️  ADVERTENCIA DE SEGURIDAD: La DSN de respaldo contiene credenciales en texto
// plano dentro del código fuente. Esto es aceptable SOLO para desarrollo local,
// pero NUNCA debe llegar a un repositorio público. Considera usar un archivo
// .env.local ignorado por .gitignore como alternativa más segura.
//
// ⚠️  BUG CONOCIDO: La variable local 'dsn' declarada dentro del bloque 'if dsn == ""'
// con ':=' es una variable NUEVA que sombrea (shadowing) a la variable exterior.
// Esto significa que la DSN local nunca llega a la llamada gorm.Open(), y si
// DATABASE_URL está vacía el servidor intentará conectarse con una cadena vacía,
// fallando silenciosamente o con un error críptico.
//
// CORRECCIÓN: Cambiar 'dsn :=' por 'dsn =' dentro del bloque if.
func ConectarDB() {

	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		// ❌ BUG: 'dsn :=' declara una variable LOCAL nueva.
		//         La variable exterior 'dsn' queda vacía.
		// ✅ FIX:  Cambiar a 'dsn =' para asignar a la variable exterior.
		dsn := "host=localhost user=postgres password=Nada123@ dbname=proyectocartones port=5432 sslmode=disable"
		// Alternativa comentada para otro entorno de desarrollo (diferente contraseña):
		// dsn = "host=localhost user=postgres password=qwerty1 dbname=proyectocartones port=5432 sslmode=disable"
		log.Println("Conectando a la base de datos LOCAL...", dsn)
	} else {
		log.Println("Conectando a la base de datos de PRODUCCIÓN (Railway)...")
	}

	var err error
	// gorm.Open abre la conexión usando el driver de PostgreSQL.
	// &gorm.Config{} usa la configuración por defecto de GORM.
	// El resultado se asigna a la variable global DB para uso en todo el proyecto.
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		// log.Fatal imprime el error y llama a os.Exit(1),
		// deteniendo completamente el servidor si no se puede conectar.
		log.Fatal("Error conectando a PostgreSQL:", err)
	}

	log.Println("✅ Conectado a PostgreSQL")
}

// MigrarModelos ejecuta las migraciones automáticas de GORM (AutoMigrate).
//
// AutoMigrate compara las structs de Go con las tablas existentes en la BD y:
//   - Crea la tabla si no existe.
//   - Agrega columnas faltantes si el modelo tiene nuevos campos.
//   - NO elimina columnas ni tablas que ya no estén en el modelo (operación segura).
//
// Orden de migración: los modelos con dependencias (foreign keys) deben migrarse
// DESPUÉS de los modelos que referencian. El orden actual es correcto:
//
//	Rol → Usuario → CartaCache → ColeccionUsuario → PublicacionVenta → Transaccion
//
// NOTA: AutoMigrate es útil en desarrollo, pero en producción se recomienda usar
// migraciones versionadas (ej. golang-migrate) para mayor control y reversibilidad.
func MigrarModelos() {
	DB.AutoMigrate(
		&models.Rol{},              // Tabla: roles — base del sistema de permisos
		&models.Usuario{},          // Tabla: usuarios — depende de Rol (foreign key)
		&models.CartaCache{},       // Tabla: cartas_caches — datos de cartas obtenidos de APIs externas
		&models.ColeccionUsuario{}, // Tabla: coleccion_usuarios — relación Usuario ↔ CartaCache
		&models.PublicacionVenta{}, // Tabla: publicacion_ventas — cartas publicadas para venta/intercambio
		&models.Transaccion{},      // Tabla: transaccions — historial de compras/ventas entre usuarios
	)
}
