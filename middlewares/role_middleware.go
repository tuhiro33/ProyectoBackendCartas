// Este archivo complementa auth_middleware.go en la cadena de seguridad:
//  1. AuthMiddleware  → verifica que el token JWT sea válido e inyecta rol_id
//  2. RequireRoles    → verifica que el rol_id inyectado tenga permiso para la ruta
//
// Uso típico en main.go:
//
//	admin := r.Group("/admin")
//	admin.Use(middlewares.AuthMiddleware())  // Paso 1: ¿está autenticado?
//	admin.Use(middlewares.RequireRoles(2))   // Paso 2: ¿tiene el rol correcto?
package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRoles devuelve un middleware de autorización basada en roles (RBAC simple).
//
// Recibe uno o más IDs de rol permitidos como argumentos variádicos,
// lo que permite proteger una misma ruta para múltiples roles sin duplicar código.
//
// Ejemplos de uso:
//
//	middlewares.RequireRoles(2)       // Solo administradores (rol_id = 2)
//	middlewares.RequireRoles(2, 3)    // Admins y moderadores
//	middlewares.RequireRoles(1, 2, 3) // Todos los roles con acceso especial
//
// Precondición: AuthMiddleware() debe ejecutarse ANTES en la cadena,
// ya que es quien inyecta "rol_id" en el contexto. Si se usa RequireRoles
// sin AuthMiddleware previo, siempre responderá con 401.
func RequireRoles(allowed ...uint) gin.HandlerFunc {
	return func(c *gin.Context) {

		// -------------------------------------------------------------------
		// PASO 1: Leer rol_id del contexto de Gin
		// Este valor fue inyectado por AuthMiddleware() al validar el JWT.
		// Si no existe, significa que AuthMiddleware no se ejecutó antes,
		// lo cual indica un error de configuración de rutas en main.go.
		// -------------------------------------------------------------------
		val, ok := c.Get("rol_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Rol no encontrado en token"})
			c.Abort()
			return
		}

		// -------------------------------------------------------------------
		// PASO 2: Convertir el valor del contexto al tipo concreto uint
		// c.Get() devuelve interface{}, por lo que hay que hacer type assertion.
		//
		// ⚠️  PUNTO FRÁGIL: Esta conversión falla si el tipo almacenado en
		// AuthMiddleware no es exactamente uint. Por ejemplo:
		//   - claims.RolID de tipo uint  → ✅ funciona
		//   - claims.RolID de tipo float64 (común si viene de JSON) → ❌ falla
		//   - claims.RolID de tipo int   → ❌ falla silenciosamente
		//
		// Si se cambia el tipo de RolID en utils.Claims, hay que actualizar
		// también este cast. El comentario original sugiere usar un switch
		// de tipos como solución más robusta:
		//
		//   var rolID uint
		//   switch v := val.(type) {
		//   case uint:    rolID = v
		//   case float64: rolID = uint(v)
		//   case int:     rolID = uint(v)
		//   default:      // responder 401
		//   }
		// -------------------------------------------------------------------
		rolID, ok := val.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Rol inválido en token"})
			c.Abort()
			return
		}

		// -------------------------------------------------------------------
		// PASO 3: Verificar si el rol del usuario está en la lista de permitidos
		// Recorre el slice 'allowed' buscando coincidencia con el rol del usuario.
		// Si encuentra una, llama c.Next() y retorna inmediatamente,
		// permitiendo que la petición continúe al controlador.
		// -------------------------------------------------------------------
		for _, a := range allowed {
			if a == rolID {
				c.Next() // ✅ Rol autorizado → continuar con el controlador
				return
			}
		}

		// -------------------------------------------------------------------
		// PASO 4: Ningún rol coincidió → acceso denegado
		// HTTP 403 Forbidden (vs 401 Unauthorized):
		//   - 401 = no autenticado (no sabemos quién eres)
		//   - 403 = autenticado pero sin permisos (sabemos quién eres, pero no puedes)
		// Esta distinción es semánticamente correcta aquí.
		// -------------------------------------------------------------------
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permisos"})
		c.Abort()
	}
}
