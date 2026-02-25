package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("clave_super_secreta_cambiar_luego")

type Claims struct {
	UserID uint `json:"user_id"`
	RolID  uint `json:"rol_id"`
	jwt.RegisteredClaims
}

func GenerarToken(userID uint, rolID uint) (string, error) {
	claims := Claims{
		UserID: userID,
		RolID:  rolID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(jwtSecret)
}

func GetJWTSecret() []byte {
	return jwtSecret
}
