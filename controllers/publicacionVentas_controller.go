package controllers

import (
	"net/http"

	"ProyectoGinBack/config"
	"ProyectoGinBack/dto"
	"ProyectoGinBack/models"

	"github.com/gin-gonic/gin"
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datos inválidos",
		})
		return
	}

	userID := c.GetUint("user_id")

	publicacion := models.PublicacionVenta{
		VendedorID:        userID,
		ColeccionID:       request.ColeccionID,
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
	userID := c.GetUint("user_id")

	var publicaciones []models.PublicacionVenta

	config.DB.
		Where("vendedor_id = ? AND estado_publicacion = ?", userID, "Activa").
		Preload("Vendedor").
		Preload("Coleccion").
		Find(&publicaciones)

	var response []dto.PublicacionResponse
	for _, p := range publicaciones {
		response = append(response, dto.MapPublicacionToDTO(p))
	}

	c.JSON(http.StatusOK, response)
}
