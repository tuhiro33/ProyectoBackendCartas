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

type CrearPublicacionRequest struct {
	VendedorID        uint    `json:"vendedor_id" binding:"required"`
	ColeccionID       uint    `json:"coleccion_id" binding:"required"`
	Precio            float64 `json:"precio" binding:"required"`
	EstadoCarta       string  `json:"estado_carta" binding:"required"`
	FotoURL           string  `json:"foto_url"`
	EstadoPublicacion string  `json:"estado_publicacion" binding:"required"`
}

type UpdatePublicacionRequest struct {
	Precio      float64 `json:"precio"`
	EstadoCarta string  `json:"estado_carta"`
	FotoURL     string  `json:"foto_url"`
}

func ObtenerPublicaciones(c *gin.Context) {
	var publicaciones []models.PublicacionVenta
	config.DB.
		Where("estado_publicacion = ?", "Activa").
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

func ObtenerPublicacionPorID(c *gin.Context) {
	id := c.Param("id")

	var publicacion models.PublicacionVenta

	result := config.DB.
		Preload("Vendedor").
		Preload("Coleccion").
		First(&publicacion, id)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Publicación no encontrada",
		})
		return
	}

	response := dto.MapPublicacionToDTO(publicacion)
	c.JSON(http.StatusOK, response)
}

func CrearPublicacion(c *gin.Context) {
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

	userID := c.GetUint("user_id")

	// 1. Obtener la entrada de colección para saber la cantidad disponible
	var coleccion models.ColeccionUsuario
	if err := config.DB.First(&coleccion, request.ColeccionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entrada de colección no encontrada"})
		return
	}

	// 2. Verificar que la colección pertenece al usuario
	if coleccion.UsuarioID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso sobre esta carta"})
		return
	}

	// 3. Contar publicaciones activas que ya existen para esta colección
	var publicacionesActivas int64
	config.DB.Model(&models.PublicacionVenta{}).
		Where("coleccion_id = ? AND estado_publicacion = ?", request.ColeccionID, "Activa").
		Count(&publicacionesActivas)

	// 4. Comparar con la cantidad disponible
	if int(publicacionesActivas) >= coleccion.Cantidad {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf(
				"Ya tienes %d publicaciones activas para esta carta. Solo tienes %d copia(s).",
				publicacionesActivas, coleccion.Cantidad,
			),
		})
		return
	}

	// 5. Todo bien, crear la publicación
	publicacion := models.PublicacionVenta{
		VendedorID:        userID,
		ColeccionID:       &request.ColeccionID,
		Precio:            request.Precio,
		EstadoCarta:       request.EstadoCarta,
		FotoURL:           request.FotoURL,
		EstadoPublicacion: "Activa",
	}
	config.DB.Create(&publicacion)
	c.JSON(http.StatusCreated, publicacion)
}

func ActualizarPublicacion(c *gin.Context) {
	// ID de la publicación desde la URL
	id := c.Param("id")

	// Usuario autenticado desde el token
	userID := c.GetUint("user_id")

	// Buscar la publicación
	var publicacion models.PublicacionVenta
	if err := config.DB.First(&publicacion, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Publicación no encontrada",
		})
		return
	}

	// Verificar que sea el dueño
	if publicacion.VendedorID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "No tienes permiso para modificar esta publicación",
		})
		return
	}

	// Bind del body
	var request UpdatePublicacionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datos inválidos",
		})
		return
	}

	// Actualizar solo campos permitidos
	publicacion.Precio = request.Precio
	publicacion.EstadoCarta = request.EstadoCarta
	publicacion.FotoURL = request.FotoURL

	// Guardar cambios
	if err := config.DB.Save(&publicacion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al actualizar la publicación",
		})
		return
	}

	// Respuesta
	c.JSON(http.StatusOK, publicacion)
}

func EliminarPublicacion(c *gin.Context) {
	// ID desde la URL
	id := c.Param("id")

	// Usuario desde el token
	userID := c.GetUint("user_id")

	// Buscar publicación
	var publicacion models.PublicacionVenta
	if err := config.DB.First(&publicacion, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Publicación no encontrada",
		})
		return
	}

	// Verificar dueño
	if publicacion.VendedorID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "No tienes permiso para eliminar esta publicación",
		})
		return
	}

	// Borrado lógico
	publicacion.EstadoPublicacion = "Eliminada"

	if err := config.DB.Save(&publicacion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al eliminar la publicación",
		})
		return
	}

	// Respuesta
	c.JSON(http.StatusOK, gin.H{
		"message": "Publicación eliminada correctamente",
	})
}

func ObtenerMisPublicaciones(c *gin.Context) {

	var publicaciones []models.PublicacionVenta
	config.DB.
		Where("estado_publicacion = ?", "Activa").
		Preload("Vendedor").
		Preload("Coleccion").
		Preload("Coleccion.Carta"). // ← agregar este
		Find(&publicaciones)

	var response []dto.PublicacionResponse
	for _, p := range publicaciones {
		response = append(response, dto.MapPublicacionToDTO(p))
	}

	c.JSON(http.StatusOK, response)
}

func MarcarComoVendida(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetUint("user_id")

	var publicacion models.PublicacionVenta
	if err := config.DB.First(&publicacion, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Publicación no encontrada"})
		return
	}

	if publicacion.VendedorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso sobre esta publicación"})
		return
	}

	if publicacion.ColeccionID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Esta publicación ya no tiene carta asociada"})
		return
	}

	var coleccion models.ColeccionUsuario
	if err := config.DB.First(&coleccion, *publicacion.ColeccionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontró la entrada de colección asociada"})
		return
	}

	// Transacción simple: Solo modificamos cantidades y estados, NADA de DELETES que rompan FKs
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Cambiar estado de la publicación a Vendida
		if err := tx.Model(&publicacion).Update("estado_publicacion", "Vendida").Error; err != nil {
			return err
		}

		// 2. Reducir la cantidad en la colección (si es 1, pasará a ser 0)
		if err := tx.Model(&coleccion).Update("cantidad", coleccion.Cantidad-1).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Printf("--- [ERROR DB] Falló MarcarComoVendida: %v ---\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error interno en la base de datos",
			"detalle": err.Error(),
		})
		return
	}

	// Responder al cliente con la cantidad nueva (que puede ser 0)
	c.JSON(http.StatusOK, gin.H{
		"message":           "Carta marcada como vendida exitosamente",
		"cantidad_restante": coleccion.Cantidad - 1,
	})
}
