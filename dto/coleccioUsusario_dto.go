// Este archivo define los DTOs de entrada (request) para el módulo de colección.
// A diferencia de usuario_dto.go que define DTOs de salida (response),
// este DTO valida y estructura los datos que llegan del cliente al servidor.
package dto

// AgregarCartaRequest es el cuerpo JSON esperado en POST /coleccion.
//
// Diseño: el DTO combina en una sola petición los datos de la entrada
// en colección (cantidad, foil) y los datos de la carta en sí.
// Esto permite al controlador hacer dos operaciones atómicas:
//  1. Verificar si la carta ya existe en CartaCache (por ApiID).
//  2. Si no existe, crearla. Si existe, reutilizarla.
//  3. Crear la entrada en ColeccionUsuario vinculando usuario y carta.
//
// Ejemplo de JSON válido:
//
//	{
//	  "usuario_id": 5,
//	  "cantidad": 2,
//	  "es_foil": true,
//	  "carta": {
//	    "api_id": "d5a1a367-550f-4c66-babc-7f6e9a8b8a2c",
//	    "juego": "magic",
//	    "nombre": "Black Lotus",
//	    "url_imagen": "https://cards.scryfall.io/normal/front/..."
//	  }
//	}
type AgregarCartaRequest struct {
	// UsuarioID identifica a qué colección agregar la carta.
	// binding:"required" hace que Gin rechace la petición con HTTP 400
	// si este campo no está presente o es 0 (valor zero de uint).
	//
	// ⚠️  CONSIDERACIÓN DE SEGURIDAD: el cliente envía su propio UsuarioID,
	// lo que permite que un usuario autenticado agregue cartas a la colección
	// de otro usuario si el controlador no valida que este ID coincida
	// con el user_id del JWT.
	// Recomendación: ignorar este campo y obtener el ID directamente
	// del contexto de Gin:
	//   usuarioID := c.GetUint("user_id") // inyectado por AuthMiddleware
	UsuarioID uint `json:"usuario_id" binding:"required"`

	// Cantidad indica cuántas copias de la carta se agregan a la colección.
	// binding:"required" rechaza la petición si falta, pero NO valida
	// que el valor sea positivo — una cantidad de 0 o negativa pasaría.
	// Para validar el rango se puede usar binding:"required,min=1":
	//   Cantidad int `json:"cantidad" binding:"required,min=1"`
	Cantidad int `json:"cantidad" binding:"required"`

	// EsFoil indica si la carta es una versión foil (holográfica/especial).
	// No tiene binding:"required" porque false es un valor válido e
	// intencional — si se pusiera required, las cartas no-foil serían
	// rechazadas al enviar false o al omitir el campo.
	EsFoil bool `json:"es_foil"`

	// Carta es un struct anónimo anidado que contiene los datos necesarios
	// para crear o encontrar el registro en la tabla CartaCache.
	// Anidar el struct aquí (en lugar de solo enviar el ApiID) garantiza
	// que si la carta no existe en la caché local, el controlador tiene
	// todos los datos para crearla sin hacer una llamada extra a la API externa.
	Carta struct {
		// ApiID es el identificador de la carta en su API de origen.
		// Es la PK de CartaCache, por eso es obligatorio.
		ApiID string `json:"api_id" binding:"required"`

		// Juego distingue entre "magic" y "pokemon".
		// Necesario porque el mismo nombre puede existir en ambos juegos.
		Juego string `json:"juego" binding:"required"`

		// Nombre y UrlImagen se usan para crear el CartaCache si no existe,
		// evitando una segunda llamada a la API externa desde el servidor.
		Nombre    string `json:"nombre"     binding:"required"`
		UrlImagen string `json:"url_imagen" binding:"required"`
	} `json:"carta" binding:"required"`
}
