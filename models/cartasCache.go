// Paquete models: structs que GORM mapea a tablas de PostgreSQL.
package models

// CartaCache representa la tabla 'carta_caches' en PostgreSQL.
//
// Propósito — caché local de APIs externas de TCG:
// En lugar de consultar Scryfall (Magic) o PokéAPI (Pokémon) en cada
// petición, los datos de cada carta se guardan aquí la primera vez
// que se sincronizan (POST /cartas/sincronizar). Las consultas
// posteriores usan este registro local, reduciendo latencia y
// dependencia de servicios externos.
//
// Relaciones:
//   - Un CartaCache puede aparecer en muchas ColeccionUsuario (one-to-many).
//   - Un CartaCache puede aparecer en muchas PublicacionVenta (one-to-many).
//
// Ciclo de vida típico:
//  1. Usuario busca una carta en el frontend.
//  2. Frontend llama POST /cartas/sincronizar con el api_id y juego.
//  3. El controlador consulta la API externa y guarda el resultado aquí.
//  4. Consultas futuras usan este registro directamente.
type CartaCache struct {
	// ApiID es la clave primaria — el identificador único de la carta
	// en su API de origen (no un ID autoincremental propio).
	// Ejemplos:
	//   Magic (Scryfall):  "d5a1a367-550f-4c66-babc-7f6e9a8b8a2c" (UUID)
	//   Pokémon (PokéAPI): "pikachu" o "base1-58" según el endpoint usado
	//
	// Usar el ID externo como PK evita duplicados si se sincroniza
	// la misma carta dos veces.
	ApiID string `json:"api_id" gorm:"primaryKey;size:100"`

	// Juego identifica de qué TCG proviene la carta.
	// Valores esperados: "magic" | "pokemon"
	// Permite filtrar colecciones y aplicar lógica específica por juego
	// (por ejemplo, campos distintos en Scryfall vs PokéAPI).
	Juego string `json:"juego" gorm:"size:100"`

	// Nombre es el nombre de la carta tal como viene de la API externa.
	// Ejemplos: "Black Lotus", "Pikachu", "Charizard ex"
	// Limitado a 150 caracteres.
	Nombre string `json:"nombre" gorm:"size:150"`

	// UrlImagen almacena la URL de la imagen oficial de la carta.
	// Proviene directamente de la API externa:
	//   Magic:   campo image_uris.normal de Scryfall
	//   Pokémon: campo images.small o images.large de PokéAPI
	//
	// Se usa en el frontend para mostrar la carta en colecciones
	// y publicaciones sin volver a llamar a la API externa.
	// Limitado a 255 caracteres (suficiente para URLs estándar).
	UrlImagen string `json:"url_imagen" gorm:"size:255"`
}
