// Paquete dto (Data Transfer Object): define las estructuras que se envían
// como respuesta JSON al cliente, separadas de los modelos internos de la BD.
//
// Por qué usar DTOs en lugar de devolver models.Usuario directamente:
//   - Seguridad: excluye campos sensibles como Password (hash bcrypt).
//   - Control: define exactamente qué campos expone la API públicamente.
//   - Desacoplamiento: el modelo puede cambiar sin romper el contrato de la API.
//   - Transformación: permite aplanar relaciones (ej: u.Rol.Nombre → "admin")
//     en lugar de anidar objetos completos con datos innecesarios.
package dto

import (
	"ProyectoGinBack/models"
	"time"
)

// UsuarioResponse es la representación pública de un usuario.
// Se usa en respuestas de: GET /me, GET /usuarios, GET /usuarios/perfil/:id
//
// Campos excluidos intencionalmente respecto a models.Usuario:
//   - Password:  nunca debe salir del servidor, ni siquiera hasheada.
//   - RolID:     el ID numérico del rol no es útil para el cliente;
//     se expone el nombre legible (Rol string) en su lugar.
type UsuarioResponse struct {
	ID            uint   `json:"id"`
	NombreUsuario string `json:"nombre_usuario"`
	Email         string `json:"email"`

	// Rol expone el nombre del rol como string plano ("admin", "usuario")
	// en lugar del objeto Rol completo. Esto aplana la relación y evita
	// exponer el ID interno del rol al cliente.
	Rol string `json:"rol"`

	FechaRegistro time.Time `json:"fecha_registro"`
	FotoPerfil    string    `json:"foto_perfil"`
}

// MapUsuarioToDTO convierte un models.Usuario (estructura interna de BD)
// a un UsuarioResponse (estructura segura para enviar al cliente).
//
// Precondición importante: el campo u.Rol debe estar precargado antes de llamar
// esta función, de lo contrario u.Rol.Nombre será una cadena vacía "".
//
// Uso correcto en el controlador:
//
//	// ✅ Con Preload — Rol.Nombre disponible
//	config.DB.Preload("Rol").First(&usuario, id)
//	c.JSON(200, dto.MapUsuarioToDTO(usuario))
//
//	// ❌ Sin Preload — Rol.Nombre será ""
//	config.DB.First(&usuario, id)
//	c.JSON(200, dto.MapUsuarioToDTO(usuario))
func MapUsuarioToDTO(u models.Usuario) UsuarioResponse {
	return UsuarioResponse{
		ID:            u.ID,
		NombreUsuario: u.NombreUsuario,
		Email:         u.Email,
		Rol:           u.Rol.Nombre, // Requiere Preload("Rol") previo en la consulta
		FechaRegistro: u.FechaRegistro,
		FotoPerfil:    u.FotoPerfil,
	}
}
