package dto

import (
	"ProyectoGinBack/models"
	"time"
)

type UsuarioResponse struct {
	ID            uint      `json:"id"`
	NombreUsuario string    `json:"nombre_usuario"`
	Email         string    `json:"email"`
	Rol           string    `json:"rol"`
	FechaRegistro time.Time `json:"fecha_registro"`
	FotoPerfil    string    `json:"foto_perfil"`
}

func MapUsuarioToDTO(u models.Usuario) UsuarioResponse {
	return UsuarioResponse{
		ID:            u.ID,
		NombreUsuario: u.NombreUsuario,
		Email:         u.Email,
		Rol:           u.Rol.Nombre,
		FechaRegistro: u.FechaRegistro,
		FotoPerfil:    u.FotoPerfil,
	}
}
