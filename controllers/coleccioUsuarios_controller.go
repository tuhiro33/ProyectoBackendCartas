// Controlador de colección: gestiona la biblioteca personal de cartas TCG de cada usuario.
// Maneja la relación entre usuarios y sus cartas (ColeccionUsuario),
// y mantiene sincronizado el caché local de cartas (CartaCache).
//
// Rutas que maneja (ver main.go):
//
//	POST   /cartas/sincronizar          → SincronizarCarta          (protegida)
//	GET    /coleccion/:usuarioId        → ObtenerColeccionUsuario   (protegida)
//	POST   /coleccion                   → AgregarAColeccion         (protegida)
//	DELETE /coleccion/:id               → EliminarDeColeccion       (protegida)
//	GET    /usuarios/coleccionistas     → ObtenerUsuariosConColeccion (pública)
package controllers

import (
	"ProyectoGinBack/config"
	"ProyectoGinBack/dto"
	"ProyectoGinBack/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ObtenerColeccionUsuario devuelve todas las cartas en la colección de un usuario,
// incluyendo los datos completos de cada carta (nombre, imagen, juego)
// mediante Preload("Carta") desde CartaCache.
//
// GET /coleccion/:usuarioId (protegida con JWT)
//
// ⚠️  CONSIDERACIÓN DE SEGURIDAD: cualquier usuario autenticado puede ver
// la colección de cualquier otro usuario pasando su ID en la URL.
// Si la colección debe ser privada, verificar que usuarioID == user_id del JWT:
//
//	userID := c.GetUint("user_id")
//	if fmt.Sprint(userID) != usuarioID { → 403 Forbidden }
//
// Si la intención es que sea pública (como un perfil de coleccionista),
// mover esta ruta fuera del grupo auth o documentarlo explícitamente.
func ObtenerColeccionUsuario(c *gin.Context) {
	usuarioID := c.Param("usuarioId") // ID del usuario cuya colección se consulta

	var coleccion []models.ColeccionUsuario

	if err := config.DB.
		Preload("Carta").                   // Carga datos de CartaCache (nombre, imagen, juego)
		Where("usuario_id = ?", usuarioID). // Filtra por el usuario solicitado
		Find(&coleccion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener la colección"})
		return
	}

	// Devuelve el modelo directamente (incluye Carta precargada).
	// A diferencia de otros endpoints, aquí no hay DTO de respuesta —
	// todos los campos de ColeccionUsuario son seguros de exponer.
	c.JSON(http.StatusOK, coleccion)
}

// AgregarAColeccion agrega una carta a la colección del usuario.
// Implementa un patrón upsert en dos pasos:
//  1. Upsert en CartaCache: garantiza que la carta existe localmente
//     antes de crear la FK desde ColeccionUsuario.
//  2. Insert en ColeccionUsuario: vincula la carta al usuario.
//
// Este diseño evita errores de FK y sincroniza el caché local con los
// datos más recientes del cliente (que los obtuvo de Scryfall/PokéAPI).
//
// POST /coleccion (protegida con JWT)
//
// ⚠️  CONSIDERACIÓN DE SEGURIDAD: usa req.UsuarioID del body en lugar del JWT.
// Un usuario podría agregar cartas a la colección de otro enviando un UsuarioID diferente.
// Corrección: ignorar req.UsuarioID y usar c.GetUint("user_id") del JWT.
func AgregarAColeccion(c *gin.Context) {
	var req dto.AgregarCartaRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// -----------------------------------------------------------------------
	// PASO 1: Upsert en CartaCache
	// Save detecta si la PK (ApiID) ya existe en la tabla:
	//   - Si NO existe → INSERT nuevo registro
	//   - Si SÍ existe → UPDATE con los datos más recientes (ej: URL de imagen actualizada)
	// Esto garantiza que la FK en ColeccionUsuario.CartaApiID siempre apunte
	// a un registro válido en carta_caches.
	// -----------------------------------------------------------------------
	cartaAuto := models.CartaCache{
		ApiID:     req.Carta.ApiID,
		Nombre:    req.Carta.Nombre,
		Juego:     req.Carta.Juego,
		UrlImagen: req.Carta.UrlImagen,
	}

	if err := config.DB.Save(&cartaAuto).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear/actualizar la referencia de la carta"})
		return
	}

	// -----------------------------------------------------------------------
	// PASO 2: Crear entrada en ColeccionUsuario
	// El Save anterior garantizó la integridad referencial, por lo que
	// este Create no fallará por FK inexistente.
	//
	// ⚠️  No verifica si ya existe una entrada para el mismo usuario + carta + foil.
	// Podría crear duplicados: dos filas con mismo UsuarioID + CartaApiID + EsFoil.
	// Alternativa: buscar entrada existente y sumar cantidad en lugar de insertar nueva.
	// -----------------------------------------------------------------------
	nuevaEntrada := models.ColeccionUsuario{
		UsuarioID:  req.UsuarioID, // ⚠️  Ver advertencia de seguridad en el doc de la función
		CartaApiID: req.Carta.ApiID,
		Cantidad:   req.Cantidad,
		EsFoil:     req.EsFoil,
	}

	if err := config.DB.Create(&nuevaEntrada).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al vincular la carta al usuario"})
		return
	}

	// Devuelve la nueva entrada creada (sin Preload, Carta quedará vacía en la respuesta)
	c.JSON(http.StatusCreated, nuevaEntrada)
}

// EliminarDeColeccion elimina una entrada específica de la colección por su ID.
// Usa hard delete (borrado físico) — el registro desaparece de la BD.
//
// DELETE /coleccion/:id (protegida con JWT)
//
// ⚠️  CONSIDERACIÓN DE SEGURIDAD: no verifica que la entrada pertenezca
// al usuario autenticado. Cualquier usuario con JWT válido podría eliminar
// entradas de la colección de otro usuario conociendo el ID.
// Corrección:
//
//	userID := c.GetUint("user_id")
//	config.DB.Where("id = ? AND usuario_id = ?", id, userID).Delete(...)
//
// ⚠️  Si ColeccionUsuario tiene registros relacionados en PublicacionVenta
// (via ColeccionID FK), este delete podría fallar por restricción de FK
// o dejar publicaciones huérfanas dependiendo de la configuración de la BD.
func EliminarDeColeccion(c *gin.Context) {
	id := c.Param("id") // ID de la entrada en coleccion_usuarios

	// Delete directo por ID. Sin Unscoped() porque ColeccionUsuario no tiene
	// campo DeletedAt (soft delete no configurado en este modelo).
	// Si se agrega DeletedAt al modelo en el futuro, este Delete pasará
	// a ser soft delete automáticamente sin cambiar este código.
	result := config.DB.Delete(&models.ColeccionUsuario{}, id)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el registro"})
		return
	}

	// RowsAffected = 0 significa que el ID no existía en la BD.
	// Sin esta verificación, un ID inexistente devolvería 200 OK silenciosamente.
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontró la carta con ese ID"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Carta eliminada de la colección con éxito"})
}

// ObtenerUsuariosConColeccion devuelve la lista de usuarios que tienen
// al menos una carta en su colección (coleccionistas activos).
// Se usa en el frontend para el buscador/sugerencias de coleccionistas.
//
// GET /usuarios/coleccionistas (pública, sin JWT)
func ObtenerUsuariosConColeccion(c *gin.Context) {
	// Struct local para la proyección SQL — solo expone id y nombre,
	// sin Email, Password ni otros datos sensibles del usuario.
	// El campo json:"nombre" mantiene compatibilidad con el frontend
	// que espera "nombre" y no "nombre_usuario".
	type UsuarioSugerencia struct {
		ID     uint   `json:"id"`
		Nombre string `json:"nombre"`
	}

	var usuarios []UsuarioSugerencia

	// JOIN entre usuarios y coleccion_usuarios para traer solo
	// los usuarios que tienen al menos una carta (cantidad > 0).
	// DISTINCT evita duplicados si un usuario tiene múltiples entradas en colección.
	// "nombre_usuario AS nombre" mapea la columna de BD al campo del struct local.
	err := config.DB.Model(&models.Usuario{}).
		Joins("JOIN coleccion_usuarios ON coleccion_usuarios.usuario_id = usuarios.id").
		Where("coleccion_usuarios.cantidad > 0").
		Distinct("usuarios.id", "usuarios.nombre_usuario AS nombre").
		Find(&usuarios).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al consultar coleccionistas activos",
			// ⚠️  err.Error() expone detalles internos de BD al cliente.
			// Eliminar "detalle" en producción, igual que en MarcarComoVendida.
			"detalle": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, usuarios)
}
