package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRoles devuelve un middleware que permite solo los rolIDs indicados.
// Ejemplo: RequireRoles(2)  -> solo rol_id == 2 (admin)
func RequireRoles(allowed ...uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, ok := c.Get("rol_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Rol no encontrado en token"})
			c.Abort()
			return
		}

		rolID, ok := val.(uint)
		if !ok {
			// si usaste otro tipo en el token, intenta convertir con switch
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Rol inválido en token"})
			c.Abort()
			return
		}

		for _, a := range allowed {
			if a == rolID {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permisos"})
		c.Abort()
	}
}
