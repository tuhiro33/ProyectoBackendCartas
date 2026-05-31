// Paquete models: structs que GORM mapea a tablas de PostgreSQL.
package models

// ColeccionUsuario representa la tabla 'coleccion_usuarios' en PostgreSQL.
//
// Es la tabla pivote (join table) entre Usuario y CartaCache,
// pero con campos propios que describen CÓMO el usuario posee la carta
// (cantidad y si es foil), lo que la convierte en una relación many-to-many
// con atributos extras — más expresiva que una join table simple.
//
// Relaciones:
//   - Pertenece a un Usuario   (many-to-one): via UsuarioID
//   - Pertenece a un CartaCache (many-to-one): via CartaApiID → CartaCache.ApiID
//
// Un usuario puede tener múltiples entradas de la misma carta
// si tiene versiones foil y no-foil por separado.
//
// Flujo típico:
//  1. POST /coleccion           → crea un registro aquí (AgregarCartaRequest DTO)
//  2. GET  /coleccion/:usuarioId → lista todos los registros de ese usuario
//                                  con Preload("Carta") para traer datos de la carta
//  3. DELETE /coleccion/:id     → elimina un registro por su ID
type ColeccionUsuario struct {
	// ID es la clave primaria autoincremental de esta entrada en la colección.
	// Permite eliminar una entrada específica (DELETE /coleccion/:id)
	// sin ambigüedad, incluso si el usuario tiene la misma carta varias veces.
	ID uint `json:"id" gorm:"primaryKey"`

	// UsuarioID es la foreign key hacia la tabla 'usuarios'.
	// Identifica al propietario de esta entrada en la colección.
	// No tiene gorm:"not null" explícito, pero debería tenerlo
	// para garantizar integridad referencial a nivel de BD.
	UsuarioID uint `json:"usuario_id"`

	// CartaApiID es la foreign key hacia CartaCache.ApiID (no hacia un ID autoincremental).
	// Usar el ApiID externo como FK en lugar de un ID propio mantiene
	// consistencia con la PK de CartaCache y evita una columna redundante.
	// size:100 debe coincidir exactamente con gorm:"size:100" en CartaCache.ApiID.
	CartaApiID string `json:"carta_api_id" gorm:"size:100"`

	// Cantidad indica cuántas copias físicas de esta carta tiene el usuario.
	// No tiene validación de rango a nivel de modelo — una cantidad 0 o negativa
	// es técnicamente posible. La validación min=1 debería estar en el DTO
	// (AgregarCartaRequest) o en el controlador.
	Cantidad int `json:"cantidad"`

	// EsFoil distingue entre la versión normal y la versión foil de una carta.
	// Esto permite que un usuario tenga dos entradas de la misma carta:
	// una foil y una no-foil, cada una con su propia cantidad.
	EsFoil bool `json:"es_foil"`

	// Carta es la asociación precargada con los datos completos de CartaCache.
	// No se almacena como columna — GORM la rellena al usar Preload("Carta").
	//
	// La anotación define la FK personalizada:
	//   foreignKey:CartaApiID  → columna local que actúa como FK
	//   references:ApiID       → columna en CartaCache que es referenciada
	//
	// Sin esta anotación explícita GORM buscaría una columna 'carta_cache_id'
	// por convención, lo que fallaría porque la PK de CartaCache es ApiID (string).
	//
	// Uso correcto en controlador:
	//   config.DB.Preload("Carta").Where("usuario_id = ?", id).Find(&coleccion)
	//   // coleccion[0].Carta.Nombre → "Black Lotus"
	//   // coleccion[0].Carta.UrlImagen → "https://..."
	Carta CartaCache `json:"carta" gorm:"foreignKey:CartaApiID;references:ApiID"`
}
