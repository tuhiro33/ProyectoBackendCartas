// Paquete utils: funciones auxiliares reutilizables en todo el proyecto.
// Este archivo centraliza la lógica de generación y validación de tokens JWT,
// evitando que la clave secreta y el algoritmo de firma estén dispersos
// en múltiples archivos.
package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtSecret es la clave simétrica usada para firmar y verificar tokens JWT
// con el algoritmo HMAC-SHA256 (HS256).
//
// ⚠️  PROBLEMA CRÍTICO DE SEGURIDAD: la clave está hardcodeada en el código.
// Cualquier persona con acceso al repositorio puede firmar tokens válidos
// y suplantar cualquier usuario, incluyendo administradores.
//
// CORRECCIÓN NECESARIA — leer la clave desde variable de entorno:
//
//	var jwtSecret []byte  // sin valor inicial hardcodeado
//
//	func init() {
//	    secret := os.Getenv("JWT_SECRET")
//	    if secret == "" {
//	        log.Fatal("JWT_SECRET no está definida en el entorno")
//	    }
//	    jwtSecret = []byte(secret)
//	}
//
// Y en el archivo .env:
//
//	JWT_SECRET=una_cadena_larga_aleatoria_de_al_menos_32_caracteres
//
// La clave debe tener mínimo 32 caracteres aleatorios para HS256.
// Se puede generar con: openssl rand -base64 32
var jwtSecret = []byte("clave_super_secreta_cambiar_luego")

// Claims define el payload del token JWT — los datos que se codifican
// dentro del token y que AuthMiddleware puede leer sin consultar la BD.
//
// Embebe jwt.RegisteredClaims para heredar los campos estándar del estándar JWT:
//   - ExpiresAt (exp): cuándo expira el token
//   - IssuedAt  (iat): cuándo fue emitido
//   - (otros como Issuer, Subject, etc., no usados aquí)
//
// UserID y RolID permiten que AuthMiddleware y RequireRoles identifiquen
// al usuario y verifiquen permisos sin hacer una consulta a la BD en cada petición.
//
// ⚠️  NOTA SOBRE TIPOS: RolID es uint aquí y en models.Usuario.
// Si en algún momento se serializa/deserializa este claim como JSON intermedio
// (ej: float64), el type assertion en RequireRoles fallará.
// Ver comentario en role_middleware.go sobre el cast val.(uint).
type Claims struct {
	UserID uint `json:"user_id"`
	RolID  uint `json:"rol_id"`
	jwt.RegisteredClaims
}

// GenerarToken crea y firma un nuevo token JWT para el usuario autenticado.
// Se llama desde los controladores de Login y Register tras verificar credenciales.
//
// El token resultante tiene este ciclo de vida:
//  1. Se genera al hacer login exitoso → se envía al cliente.
//  2. El cliente lo almacena (localStorage, memoria, cookie).
//  3. El cliente lo envía en cada petición como: Authorization: Bearer <token>
//  4. AuthMiddleware lo valida y extrae UserID y RolID del payload.
//  5. A las 24 horas expira → el cliente debe volver a hacer login.
//
// Parámetros:
//   - userID: ID del usuario en la tabla 'usuarios' (models.Usuario.ID)
//   - rolID:  ID del rol del usuario (models.Usuario.RolID)
//
// Retorna el token como string firmado, o un error si la firma falla.
func GenerarToken(userID uint, rolID uint) (string, error) {
	claims := Claims{
		UserID: userID,
		RolID:  rolID,
		RegisteredClaims: jwt.RegisteredClaims{
			// El token expira 24 horas después de su emisión.
			// Para mayor seguridad se podría reducir a 1-2 horas
			// y complementar con un refresh token de larga duración.
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			// IssuedAt permite saber cuándo fue emitido,
			// útil para auditorías o para invalidar tokens emitidos
			// antes de un cambio de contraseña.
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}

	// jwt.NewWithClaims crea el token sin firmar con el algoritmo HS256.
	// HS256 (HMAC-SHA256) es simétrico: usa la misma clave para firmar y verificar.
	// Alternativa más segura para producción: RS256 (asimétrico), que permite
	// verificar tokens sin exponer la clave privada de firma.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// SignedString firma el token con jwtSecret y retorna la cadena final
	// en formato: header.payload.signature (las tres partes en base64url)
	return token.SignedString(jwtSecret)
}

// GetJWTSecret expone la clave secreta para que AuthMiddleware pueda
// verificar la firma de tokens entrantes.
//
// Centralizar el acceso aquí garantiza que tanto GenerarToken como
// AuthMiddleware usen exactamente la misma clave — si difieren,
// todos los tokens serían inválidos.
//
// Con la corrección propuesta (leer de os.Getenv), esta función
// seguiría funcionando igual para los consumidores externos:
//
//	func GetJWTSecret() []byte {
//	    return jwtSecret  // ya inicializado en init()
//	}
func GetJWTSecret() []byte {
	return jwtSecret
}
