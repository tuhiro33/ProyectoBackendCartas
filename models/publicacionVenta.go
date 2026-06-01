// Paquete models: structs que GORM mapea a tablas de PostgreSQL.
package models

import "time"

// PublicacionVenta representa la tabla 'publicacion_ventas' en PostgreSQL.
//
// Es el recurso central del marketplace — conecta a un vendedor con
// una carta de su colección y la ofrece a otros usuarios.
//
// Relaciones:
//   - Pertenece a un Usuario (vendedor)     via VendedorID  (many-to-one)
//   - Pertenece a un ColeccionUsuario       via ColeccionID (many-to-one, nullable)
//
// Ciclo de vida (EstadoPublicacion):
//   "Activa"    → visible en el marketplace, disponible para comprar
//   "Vendida"   → marcada por el vendedor al concretar una venta
//   "Eliminada" → soft delete, no visible pero el registro persiste en la BD
//
// El soft delete (cambiar estado en lugar de borrar) preserva el historial
// y evita romper foreign keys en Transaccion que apunten a esta publicación.
type PublicacionVenta struct {
	// ID es la clave primaria autoincremental.
	ID uint `gorm:"primaryKey"`

	// VendedorID es la foreign key hacia la tabla 'usuarios'.
	// Identifica quién publicó la carta para venta.
	// Se toma del JWT en CrearPublicacion — el cliente no lo envía.
	VendedorID uint

	// ColeccionID es la foreign key hacia 'coleccion_usuarios'.
	// Es un puntero (*uint) para permitir valores NULL en la BD —
	// un puntero nil en Go se traduce a NULL en PostgreSQL.
	//
	// Es nullable porque cuando la carta se marca como vendida,
	// ColeccionID podría quedar en NULL si la entrada de colección
	// se elimina, sin romper el registro de la publicación.
	// MarcarComoVendida verifica que no sea nil antes de desreferenciarlo
	// con *publicacion.ColeccionID para evitar panic en tiempo de ejecución.
	ColeccionID *uint

	// Precio almacena el valor de venta con precisión decimal.
	// decimal(10,2) permite hasta 99,999,999.99 — suficiente para
	// cualquier carta TCG. Sin este tipo GORM usaría float8 de PostgreSQL
	// que puede tener imprecisión en decimales (ej: 9.99 → 9.990000001).
	Precio float64 `gorm:"type:decimal(10,2)"`

	// EstadoCarta describe la condición física de la carta.
	// Valores estándar del mercado TCG:
	//   "NM"  = Near Mint      (casi perfecta)
	//   "LP"  = Lightly Played (leve desgaste)
	//   "MP"  = Moderately Played
	//   "HP"  = Heavily Played
	//   "DMG" = Damaged        (dañada)
	EstadoCarta string `gorm:"size:50"`

	// FotoURL almacena la URL de la imagen real de la carta subida por el vendedor
	// (guardada en Firebase Storage via /upload).
	// size:512 permite URLs más largas que el estándar 255 —
	// las URLs de Firebase Storage pueden ser extensas.
	// json:"foto_url" define el nombre del campo en respuestas JSON.
	FotoURL string `gorm:"size:512" json:"foto_url"`

	// EstadoPublicacion controla la visibilidad en el marketplace.
	// Ver ciclo de vida en el doc del struct.
	// ⚠️  No hay restricción a nivel de BD que limite los valores
	// a "Activa", "Vendida" o "Eliminada" — cualquier string pasaría.
	// Un CHECK constraint en PostgreSQL o una validación en el controlador
	// lo haría más robusto.
	EstadoPublicacion string `gorm:"size:50"`

	// FechaPublicacion se establece automáticamente por GORM al crear el registro.
	// autoCreateTime equivale a DEFAULT NOW() en SQL.
	// Se usa en el frontend para mostrar "Publicado el: 15/3/2024".
	FechaPublicacion time.Time `gorm:"autoCreateTime"`

	// Vendedor es la asociación precargada con los datos del usuario vendedor.
	// No se almacena como columna — GORM la rellena con Preload("Vendedor").
	// Permite acceder a vendedor.NombreUsuario, vendedor.Email, etc.
	// sin hacer una consulta separada.
	Vendedor Usuario `gorm:"foreignKey:VendedorID"`

	// Coleccion es la asociación precargada con la entrada de colección
	// que contiene los datos de la carta (via Coleccion.Carta con Preload anidado).
	// No se almacena como columna — se rellena con Preload("Coleccion").
	// Para obtener el nombre de la carta:
	//   Preload("Coleccion.Carta") → publicacion.Coleccion.Carta.Nombre
	Coleccion ColeccionUsuario `gorm:"foreignKey:ColeccionID"`
}
