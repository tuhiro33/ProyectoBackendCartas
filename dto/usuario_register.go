package dto

type RegisterRequest struct {
	NombreUsuario string `json:"nombre_usuario"`
	Email         string `json:"email"`
	Password      string `json:"password"`
}
