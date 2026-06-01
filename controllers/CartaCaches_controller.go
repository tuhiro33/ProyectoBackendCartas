// Controlador de caché de cartas: gestiona la sincronización de datos
// de cartas TCG desde APIs externas (Scryfall/PokéAPI) hacia la base
// de datos local en la tabla 'carta_caches'.
//
// El propósito del caché es evitar llamar a las APIs externas en cada
// petición — una vez que una carta se sincroniza, sus datos quedan
// guardados localmente y se reutilizan en colecciones y publicaciones.
//
// Ruta que maneja (ver main.go):
//
//	POST /cartas/sincronizar → SincronizarCarta (protegida con JWT)
package controllers

import (
	"ProyectoGinBack/config"
	"ProyectoGinBack/dto"
	"ProyectoGinBack/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SincronizarCarta guarda una carta nueva o actualiza sus datos si ya existe.
// Implementa el patrón "upsert" (update + insert):
//   - Si la carta NO existe en carta_caches → la inserta (INSERT)
//   - Si la carta YA existe (mismo ApiID)  → actualiza sus datos (UPDATE)
//
// POST /cartas/sincronizar (protegida con JWT)
//
// Este endpoint es llamado desde AgregarCartas.tsx en el frontend
// cuando el usuario confirma agregar una carta a su colección.
// El frontend envía los datos que obtuvo de Scryfall o TCGDex
// para que el backend los guarde localmente.
//
// ⚠️  PROBLEMA: el campo UrlImagen del modelo CartaCache no se incluye
// en el upsert — la variable 'carta' que se pasa a Assign() no tiene
// UrlImagen porque el DTO CartaCacheDTO tampoco lo tiene (ver más abajo).
// Esto significa que la imagen nunca se guarda desde este endpoint.
// La imagen SÍ se guarda correctamente desde AgregarAColeccion en
// coleccionUsuarios_controller.go que usa config.DB.Save() directamente.
func SincronizarCarta(c *gin.Context) {
	var input dto.CartaCacheDTO

	// ShouldBindJSON deserializa el body JSON en CartaCacheDTO.
	// Si falta un campo con binding:"required" o el JSON es inválido,
	// devuelve error y el handler responde con 400.
	if err := c.ShouldBindJSON(&input); err != nil {
		// ⚠️  err.Error() expone detalles internos de validación al cliente.
		// En producción usar: gin.H{"error": "Datos inválidos"}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Construye el modelo con los datos del DTO.
	// ⚠️  UrlImagen no se incluye aquí — ver advertencia en el doc de la función.
	carta := models.CartaCache{
		ApiID:  input.ApiID,
		Juego:  input.Juego,
		Nombre: input.Nombre,
	}

	// Upsert con GORM usando la combinación Where + Assign + FirstOrCreate:
	//
	//   Where(models.CartaCache{ApiID: input.ApiID})
	//     → Busca un registro donde api_id coincida con el input.
	//       Usa una struct como condición en lugar de string SQL para
	//       mayor seguridad (evita SQL injection) y legibilidad.
	//
	//   Assign(carta)
	//     → Define los valores a asignar al registro encontrado o nuevo.
	//       Si el registro existe, GORM actualiza estos campos.
	//       Si no existe, los usa para crear el nuevo registro.
	//
	//   FirstOrCreate(&carta)
	//     → Ejecuta la operación:
	//       - Si encuentra el registro → lo actualiza con Assign() y lo carga en &carta
	//       - Si no encuentra → crea uno nuevo con Where() + Assign() y lo carga en &carta
	//
	// El resultado en &carta siempre tiene el registro final (nuevo o actualizado)
	// con todos sus campos, incluyendo los que ya existían en la BD.
	result := config.DB.Where(models.CartaCache{ApiID: input.ApiID}).
		Assign(carta).
		FirstOrCreate(&carta)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al cachear carta"})
		return
	}

	// Devuelve el registro final tal como quedó en la BD.
	// Si fue un INSERT, devuelve la carta recién creada.
	// Si fue un UPDATE, devuelve la carta con los datos actualizados.
	c.JSON(http.StatusOK, carta)
}
