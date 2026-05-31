// Paquete models: define las structs que representan las tablas de la base de datos.
// GORM usa estas structs para crear tablas (AutoMigrate) y para mapear
// resultados de consultas a objetos Go.
package models

import "time"

// Usuario representa la tabla 'usuarios' en PostgreSQL.
//
// Relaciones:
//   - Pertenece a un Rol (many-to-one): cada usuario tiene exactamente un rol.
//   - Un Rol puede tener muchos Usuarios (definido en el modelo Rol).
//
// Flujo típico:
//  1. El usuario se registra vía POST /register → se crea un registro aquí.
//  2. Al hacer login, se genera un JWT con ID y RolID de este struct.
//  3. AuthMiddleware valida el JWT y usa estos IDs para autorizar peticiones.
type Usuario struct {
	// ID es la clave primaria autoincremental.
	// GORM la gestiona automáticamente al insertar un nuevo registro.
	ID uint `gorm:"primaryKey"`

	// RolID es la foreign key que vincula al usuario con la tabla 'roles'.
	// Define qué puede hacer el usuario en el sistema (ej: rol 1 = usuario, rol 2 = admin).
	// Se inyecta en el JWT y lo lee RequireRoles() para controlar acceso a rutas.
	RolID uint

	// NombreUsuario es el nombre público visible del usuario en el marketplace.
	// Limitado a 100 caracteres, obligatorio.
	NombreUsuario string `gorm:"size:100;not null"`

	// Email es el identificador único de login.
	// La restricción 'unique' en la BD previene registros duplicados.
	// Limitado a 150 caracteres, obligatorio.
	Email string `gorm:"size:150;unique;not null"`

	// Password almacena el hash de la contraseña (nunca en texto plano).
	// Se espera un hash bcrypt, que produce cadenas de ~60 caracteres,
	// por eso el tamaño es 255 para tener margen con otros algoritmos.
	//
	// ⚠️  IMPORTANTE: Este campo NO tiene la etiqueta json:"-".
	//     Esto significa que si algún controlador devuelve un Usuario
	//     directamente como JSON, el hash de la contraseña quedará expuesto.
	//     Se recomienda agregar `json:"-"` o usar un DTO de respuesta
	//     que excluya este campo.
	Password string `gorm:"size:255;not null"`

	// FechaRegistro se establece automáticamente por GORM al crear el registro.
	// 'autoCreateTime' equivale a DEFAULT NOW() en SQL.
	// No se actualiza en ediciones posteriores (para eso sería autoUpdateTime).
	FechaRegistro time.Time `gorm:"autoCreateTime"`

	// FotoPerfil almacena la URL de la imagen de perfil del usuario
	// (por ejemplo, una URL de Cloudinary o del endpoint /upload).
	// Es opcional (sin 'not null'), por eso puede ser cadena vacía.
	// La etiqueta json:"foto_perfil" define el nombre del campo en la respuesta JSON.
	FotoPerfil string `gorm:"size:255" json:"foto_perfil"`

	// Rol es la asociación precargada del rol del usuario.
	// GORM usa 'foreignKey:RolID' para hacer JOIN con la tabla 'roles'
	// cuando se usa Preload("Rol") en una consulta.
	// Este campo NO se almacena como columna — es solo para las consultas con JOIN.
	//
	// Ejemplo de uso en controlador:
	//   db.Preload("Rol").First(&usuario, id)
	//   fmt.Println(usuario.Rol.Nombre) // "admin", "usuario", etc.
	Rol Rol `gorm:"foreignKey:RolID"`
}
