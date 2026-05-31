// Controlador de publicaciones: gestiona las cartas TCG publicadas para venta
// en el marketplace entre usuarios.
//
// Rutas que maneja (ver main.go):
//
//	GET    /publicaciones              → ObtenerPublicaciones     (pública)
//	GET    /publicaciones/:id          → ObtenerPublicacionPorID  (pública)
//	POST   /publicaciones              → CrearPublicacion         (protegida)
//	PUT    /publicaciones/:id          → ActualizarPublicacion    (protegida)
//	DELETE /publicaciones/:id          → EliminarPublicacion      (protegida)
//	GET    /mis-publicaciones          → ObtenerMisPublicaciones  (protegida)
//	PUT    /publicaciones/:id/vendida  → MarcarComoVendida        (protegida)
package controllers

import (
	"fmt"
	"net/http"

	"ProyectoGinBack/config"
	"ProyectoGinBack/dto"
	"ProyectoGinBack/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CrearPublicacionRequest es el DTO de entrada para el endpoint POST /publicaciones.
// Definido a nivel de paquete pero con campos que ya no coinciden con
// la implementación actual de CrearPublicacion (que usa un struct anónimo).
//
// ⚠️  Este struct no se usa en ninguna función de este archivo —
// CrearPublicacion define su propio struct anónimo internamente.
// Considerar eliminar este struct o unificarlo con el anónimo interno.
type CrearPublicacionRequest struct {
	VendedorID        uint    `json:"vendedor_id" binding:"required"`
	ColeccionID       uint    `json:"coleccion_id" binding:"required"`
	Precio            float64 `json:"precio" binding:"required"`
	EstadoCarta       string  `json:"estado_carta" binding:"required"`
	FotoURL           string  `json:"foto_url"`
	EstadoPublicacion string  `json:"estado_publicacion" binding:"required"`
}

// UpdatePublicacionRequest define los campos editables de una publicación.
// Todos son opcionales — el vendedor puede actualizar precio, estado de la
// carta o foto sin tocar los demás campos.
//
// ⚠️  A diferencia de ActualizarUsuario, aquí NO se verifica si el valor
// está vacío antes de asignar. Si el cliente envía precio:0 o estado_carta:"",
// esos valores se guardarán en la BD. Considerar validaciones mínimas:
//
//	Precio float64 `json:"precio" binding:"omitempty,min=0"`
type UpdatePublicacionRequest struct {
	Precio      float64 `json:"precio"`
	EstadoCarta string  `json:"estado_carta"`
	FotoURL     string  `json:"foto_url"`
}

// ObtenerPublicaciones devuelve todas las publicaciones con estado "Activa".
// Carga en cascada: Vendedor → Coleccion → Coleccion.Carta (CartaCache)
// para que el DTO pueda armar una respuesta completa con datos de la carta.
//
// GET /publicaciones (pública, sin JWT)
//
// ⚠️  Sin paginación ni filtros: carga todas las publicaciones activas en memoria.
// Con muchas publicaciones esto puede ser lento. Considerar agregar:
//   - Paginación:  ?page=1&limit=20
//   - Filtros:     ?juego=magic&precio_max=50
func ObtenerPublicaciones(c *gin.Context) {
	var publicaciones []models.PublicacionVenta
	config.DB.
		Where("estado_publicacion = ?", "Activa").
		Preload("Vendedor").        // Datos del usuario vendedor
		Preload("Coleccion").       // Entrada de colección (cantidad, es_foil)
		Preload("Coleccion.Carta"). // Datos de la carta (nombre, imagen, juego)
		Find(&publicaciones)

	// Convertir al DTO de respuesta para controlar qué campos se exponen
	var response []dto.PublicacionResponse
	for _, p := range publicaciones {
		response = append(response, dto.MapPublicacionToDTO(p))
	}

	// ⚠️  Si no hay publicaciones activas, response es nil → se serializa como null.
	// Para devolver [] usar: response := make([]dto.PublicacionResponse, 0)
	c.JSON(http.StatusOK, response)
}

// ObtenerPublicacionPorID devuelve el detalle de una publicación específica.
// No filtra por estado — devuelve también publicaciones vendidas o eliminadas,
// lo que puede ser útil para mostrar el historial pero conviene documentarlo.
//
// GET /publicaciones/:id (pública, sin JWT)
func ObtenerPublicacionPorID(c *gin.Context) {
	id := c.Param("id")

	var publicacion models.PublicacionVenta
	result := config.DB.
		Preload("Vendedor").
		Preload("Coleccion").
		// ⚠️  Falta Preload("Coleccion.Carta") presente en ObtenerPublicaciones.
		// Si MapPublicacionToDTO intenta acceder a Coleccion.Carta.Nombre,
		// devolverá cadena vacía en este endpoint.
		First(&publicacion, id)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Publicación no encontrada",
		})
		return
	}

	c.JSON(http.StatusOK, dto.MapPublicacionToDTO(publicacion))
}

// CrearPublicacion publica una carta de la colección del usuario para venta.
//
// POST /publicaciones (protegida con JWT)
//
// Lógica de integridad — antes de crear verifica:
//  1. La entrada de colección existe.
//  2. Pertenece al usuario autenticado (no a otro).
//  3. No hay más publicaciones activas que copias disponibles.
//     Ej: si el usuario tiene 2 copias, puede tener máximo 2 publicaciones activas.
//
// Esto evita que se vendan más cartas de las que el usuario realmente posee.
func CrearPublicacion(c *gin.Context) {
	// Struct anónimo en lugar de CrearPublicacionRequest (definido arriba pero no usado).
	// VendedorID no está aquí — se toma del JWT para evitar que el cliente
	// se asigne publicaciones a nombre de otro usuario.
	var request struct {
		ColeccionID uint    `json:"coleccion_id"`
		Precio      float64 `json:"precio"`
		EstadoCarta string  `json:"estado_carta"`
		FotoURL     string  `json:"foto_url"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// El ID del vendedor viene del JWT — no del body del cliente.
	// Esto garantiza que nadie puede crear publicaciones a nombre de otro usuario.
	userID := c.GetUint("user_id")

	// PASO 1: Verificar que la entrada de colección existe
	var coleccion models.ColeccionUsuario
	if err := config.DB.First(&coleccion, request.ColeccionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entrada de colección no encontrada"})
		return
	}

	// PASO 2: Verificar propiedad — la colección debe pertenecer al usuario del JWT
	if coleccion.UsuarioID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso sobre esta carta"})
		return
	}

	// PASO 3: Contar publicaciones activas existentes para esta entrada de colección
	var publicacionesActivas int64
	config.DB.Model(&models.PublicacionVenta{}).
		Where("coleccion_id = ? AND estado_publicacion = ?", request.ColeccionID, "Activa").
		Count(&publicacionesActivas)

	// PASO 4: No permitir más publicaciones activas que copias disponibles
	if int(publicacionesActivas) >= coleccion.Cantidad {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf(
				"Ya tienes %d publicaciones activas para esta carta. Solo tienes %d copia(s).",
				publicacionesActivas, coleccion.Cantidad,
			),
		})
		return
	}

	// PASO 5: Crear la publicación con estado inicial "Activa"
	// EstadoPublicacion se fuerza a "Activa" desde el servidor —
	// el cliente no puede crear publicaciones con estado arbitrario.
	publicacion := models.PublicacionVenta{
		VendedorID:        userID,
		ColeccionID:       &request.ColeccionID, // puntero porque es nullable en el modelo
		Precio:            request.Precio,
		EstadoCarta:       request.EstadoCarta,
		FotoURL:           request.FotoURL,
		EstadoPublicacion: "Activa",
	}
	config.DB.Create(&publicacion)

	// ⚠️  Devuelve models.PublicacionVenta directamente sin DTO.
	// Si el modelo tiene campos sensibles o relaciones no cargadas,
	// la respuesta puede ser incompleta o inconsistente.
	// Recomendación: usar dto.MapPublicacionToDTO(publicacion)
	c.JSON(http.StatusCreated, publicacion)
}

// ActualizarPublicacion permite al vendedor modificar precio, estado y foto
// de una publicación que le pertenece.
//
// PUT /publicaciones/:id (protegida con JWT)
//
// ⚠️  Sobreescribe los campos sin verificar valores vacíos:
// si el cliente envía {"precio": 0}, el precio se guarda como 0.
// Considerar el mismo patrón de ActualizarUsuario (solo actualizar si != zero value).
func ActualizarPublicacion(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetUint("user_id") // Del JWT — para verificar propiedad

	var publicacion models.PublicacionVenta
	if err := config.DB.First(&publicacion, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Publicación no encontrada",
		})
		return
	}

	// Verificar que el usuario autenticado es el dueño de la publicación
	if publicacion.VendedorID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "No tienes permiso para modificar esta publicación",
		})
		return
	}

	var request UpdatePublicacionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datos inválidos",
		})
		return
	}

	// Asignación directa sin verificar zero values — ver advertencia en UpdatePublicacionRequest
	publicacion.Precio = request.Precio
	publicacion.EstadoCarta = request.EstadoCarta
	publicacion.FotoURL = request.FotoURL

	if err := config.DB.Save(&publicacion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al actualizar la publicación",
		})
		return
	}

	// ⚠️  Devuelve el modelo directamente — considerar DTO para consistencia
	c.JSON(http.StatusOK, publicacion)
}

// EliminarPublicacion implementa un soft delete — cambia el estado a "Eliminada"
// en lugar de borrar el registro físicamente.
//
// DELETE /publicaciones/:id (protegida con JWT)
//
// Ventaja del soft delete: preserva el historial de publicaciones y evita
// romper foreign keys en tablas relacionadas (Transaccion, etc.).
func EliminarPublicacion(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetUint("user_id")

	var publicacion models.PublicacionVenta
	if err := config.DB.First(&publicacion, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Publicación no encontrada",
		})
		return
	}

	// Solo el vendedor propietario puede eliminar su publicación
	if publicacion.VendedorID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "No tienes permiso para eliminar esta publicación",
		})
		return
	}

	// Soft delete: cambiar estado a "Eliminada" en lugar de DELETE FROM BD.
	// Las publicaciones eliminadas no aparecen en ObtenerPublicaciones
	// (que filtra por estado_publicacion = 'Activa') pero el registro persiste.
	publicacion.EstadoPublicacion = "Eliminada"

	if err := config.DB.Save(&publicacion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al eliminar la publicación",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Publicación eliminada correctamente",
	})
}

// ObtenerMisPublicaciones devuelve las publicaciones activas del usuario autenticado.
//
// GET /mis-publicaciones (protegida con JWT)
//
// ⚠️  BUG: no filtra por el usuario del JWT — devuelve TODAS las publicaciones
// activas del sistema, igual que ObtenerPublicaciones.
// Falta agregar: Where("vendedor_id = ?", userID)
//
// Corrección:
//
//	userID := c.GetUint("user_id")
//	config.DB.
//	    Where("estado_publicacion = ? AND vendedor_id = ?", "Activa", userID).
//	    Preload(...)
func ObtenerMisPublicaciones(c *gin.Context) {
	// ❌ userID nunca se lee del contexto
	// userID := c.GetUint("user_id")

	var publicaciones []models.PublicacionVenta
	config.DB.
		Where("estado_publicacion = ?", "Activa").
		// ❌ Falta: Where("vendedor_id = ?", userID)
		Preload("Vendedor").
		Preload("Coleccion").
		Preload("Coleccion.Carta").
		Find(&publicaciones)

	var response []dto.PublicacionResponse
	for _, p := range publicaciones {
		response = append(response, dto.MapPublicacionToDTO(p))
	}

	c.JSON(http.StatusOK, response)
}

// MarcarComoVendida registra que una carta fue vendida y actualiza
// el inventario del vendedor de forma atómica.
//
// PUT /publicaciones/:id/vendida (protegida con JWT)
//
// Usa una transacción de BD para garantizar que ambas operaciones
// (cambiar estado + reducir cantidad) ocurran juntas o ninguna ocurra.
// Esto evita inconsistencias como: publicación marcada como vendida
// pero cantidad de colección sin reducir (o viceversa).
func MarcarComoVendida(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetUint("user_id")

	var publicacion models.PublicacionVenta
	if err := config.DB.First(&publicacion, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Publicación no encontrada"})
		return
	}

	// Solo el vendedor puede marcar su propia publicación como vendida
	if publicacion.VendedorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso sobre esta publicación"})
		return
	}

	// ColeccionID es un puntero nullable — verificar que no sea nil
	// antes de desreferenciarlo para evitar panic en tiempo de ejecución.
	if publicacion.ColeccionID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Esta publicación ya no tiene carta asociada"})
		return
	}

	var coleccion models.ColeccionUsuario
	if err := config.DB.First(&coleccion, *publicacion.ColeccionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontró la entrada de colección asociada"})
		return
	}

	// Transacción atómica: las dos operaciones se confirman juntas (commit)
	// o se deshacen juntas (rollback) si cualquiera falla.
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// OPERACIÓN 1: Marcar publicación como vendida
		// Usa Update en lugar de Save para modificar solo este campo,
		// evitando condiciones de carrera con otras actualizaciones concurrentes.
		if err := tx.Model(&publicacion).Update("estado_publicacion", "Vendida").Error; err != nil {
			return err // Rollback automático al retornar error
		}

		// OPERACIÓN 2: Reducir cantidad en la colección del vendedor.
		// Si la cantidad llega a 0 el registro persiste en la BD
		// (no se elimina) para mantener el historial y respetar las FK.
		if err := tx.Model(&coleccion).Update("cantidad", coleccion.Cantidad-1).Error; err != nil {
			return err // Rollback automático
		}

		return nil // Commit: ambas operaciones exitosas
	})

	if err != nil {
		// Log del error real en servidor (no se expone al cliente por seguridad)
		fmt.Printf("--- [ERROR DB] Falló MarcarComoVendida: %v ---\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error interno en la base de datos",
			// ⚠️  err.Error() expone detalles internos de BD al cliente.
			// En producción eliminar "detalle" de la respuesta JSON.
			"detalle": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Carta marcada como vendida exitosamente",
		// cantidad_restante se calcula en memoria (coleccion.Cantidad - 1)
		// porque la transacción ya actualizó la BD pero no refrescó el struct local.
		// El valor es correcto pero viene del struct en memoria, no de un SELECT posterior.
		"cantidad_restante": coleccion.Cantidad - 1,
	})
}
