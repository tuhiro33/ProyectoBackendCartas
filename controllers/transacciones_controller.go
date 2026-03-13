package controllers

import (
	"ProyectoGinBack/config"
	"ProyectoGinBack/dto"
	"ProyectoGinBack/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CrearTransaccion(c *gin.Context) {
	var input dto.CrearTransaccionDTO

	// Validar entrada
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mapear al modelo
	transaccion := models.Transaccion{
		PublicacionID: input.PublicacionID,
		CompradorID:   input.CompradorID,
		PrecioFinal:   input.PrecioFinal,
		EstadoPago:    input.EstadoPago,
	}

	// Guardar transacción
	if err := config.DB.Create(&transaccion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo registrar la transacción"})
		return
	}

	// Aquí cambiamos el estado de la publicación a "vendido" automáticamente
	config.DB.Model(&models.PublicacionVenta{}).
		Where("id = ?", input.PublicacionID).
		Update("estado_publicacion", "vendido")

	c.JSON(http.StatusCreated, transaccion)
}

func ObtenerHistorialCompras(c *gin.Context) {
	usuarioID := c.Param("usuarioId")
	var transacciones []models.Transaccion

	// Usamos Preload para traer los datos de la carta vendida en la misma consulta
	result := config.DB.Preload("Publicacion").Where("comprador_id = ?", usuarioID).Find(&transacciones)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener historial"})
		return
	}

	c.JSON(http.StatusOK, transacciones)
}
