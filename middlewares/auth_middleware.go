// Paquete middlewares: contiene las funciones intermediarias que se ejecutan
// antes de que una petición llegue al controlador final.
// Este archivo implementa la autenticación mediante tokens JWT.
package middlewares

import (
	"net/http"
	"strings"

	"ProyectoGinBack/utils" // Contiene Claims y GetJWTSecret()

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware es un middleware de Gin que protege rutas mediante autenticación JWT.
//
// Flujo de validación:
//  1. Extrae el header "Authorization" de la petición HTTP.
//  2. Verifica que tenga el formato "Bearer <token>".
//  3. Parsea y valida el token JWT (firma, expiración, estructura).
//  4. Si es válido, inyecta user_id y rol_id en el contexto de Gin
//     para que los controladores puedan acceder a ellos.
//  5. Si cualquier paso falla, responde con HTTP 401 y corta la cadena
//     de middlewares con c.Abort() — el controlador nunca se ejecuta.
//
// Uso en rutas:
//
//	auth := r.Group("/")
//	auth.Use(middlewares.AuthMiddleware())
//	auth.GET("/perfil", controllers.GetProfile)
//
// Uso en controladores para leer los datos inyectados:
//
//	userID := c.GetUint("user_id")
//	rolID  := c.GetUint("rol_id")
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// -----------------------------------------------------------------------
		// PASO 1: Verificar que el header Authorization existe
		// El cliente debe enviar: Authorization: Bearer eyJhbGciOiJ...
		// -----------------------------------------------------------------------
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token requerido",
			})
			// c.Abort() detiene la ejecución de todos los handlers siguientes
			// en la cadena (otros middlewares y el controlador final).
			// Sin Abort(), Gin continuaría ejecutando el controlador aunque
			// hayamos respondido con 401.
			c.Abort()
			return
		}

		// -----------------------------------------------------------------------
		// PASO 2: Validar formato "Bearer <token>"
		// Split divide por espacio: ["Bearer", "eyJhbGci..."]
		// Debe tener exactamente 2 partes y la primera debe ser "Bearer".
		// Formatos inválidos: solo el token, "Token xxx", "bearer xxx" (case-sensitive).
		// -----------------------------------------------------------------------
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Formato de token inválido",
			})
			c.Abort()
			return
		}

		tokenStr := parts[1] // La cadena JWT en sí: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

		// -----------------------------------------------------------------------
		// PASO 3: Parsear y validar el token JWT
		// jwt.ParseWithClaims hace tres cosas simultáneamente:
		//   a) Decodifica el payload del token en la struct Claims.
		//   b) Verifica la firma usando la clave secreta (GetJWTSecret).
		//   c) Valida claims estándar: expiración (exp), emisión (iat), etc.
		//
		// La función callback recibe el token sin verificar y debe devolver
		// la clave secreta con la que se firmó. Aquí se usa una función
		// centralizada en utils para no duplicar la clave en múltiples lugares.
		// -----------------------------------------------------------------------
		claims := &utils.Claims{} // Struct personalizada que contiene UserID y RolID

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			// GetJWTSecret() lee JWT_SECRET del entorno (.env o variable de sistema).
			// Devolver la clave aquí permite que la librería verifique la firma HMAC.
			return utils.GetJWTSecret(), nil
		})

		// El token es inválido si:
		//   - La firma no coincide con la clave secreta (token manipulado o clave incorrecta)
		//   - El token está expirado (claim "exp" en el pasado)
		//   - El formato del JWT está malformado
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token inválido",
			})
			c.Abort()
			return
		}

		// -----------------------------------------------------------------------
		// PASO 4: Inyectar datos del usuario en el contexto de Gin
		// c.Set almacena valores clave-valor que viajan con la petición
		// y son accesibles en cualquier handler posterior con c.Get().
		//
		// user_id → identifica al usuario propietario de la petición
		// rol_id  → permite verificar permisos en controladores o en
		//           el middleware RequireRoles() que puede venir después
		// -----------------------------------------------------------------------
		c.Set("user_id", claims.UserID)
		c.Set("rol_id", claims.RolID)

		// c.Next() pasa el control al siguiente middleware o controlador en la cadena.
		// Si no se llama, la petición queda colgada (aunque aquí Gin lo maneja igual
		// al final del handler, es buena práctica llamarlo explícitamente).
		c.Next()
	}
}
