package controllers

import (
	"ProyectoGinBack/config"
	"ProyectoGinBack/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DTO para validar la entrada
type AgregarCartaRequest struct {
	UsuarioID  uint   `json:"usuario_id" binding:"required"`
	CartaApiID string `json:"carta_api_id" binding:"required"`
	Cantidad   int    `json:"cantidad" binding:"required"`
	EsFoil     bool   `json:"es_foil"`
}

// Obtener toda la colección de un usuario
func ObtenerColeccionUsuario(c *gin.Context) {
	usuarioID := c.Param("usuarioId")
	var coleccion []models.ColeccionUsuario

	if err := config.DB.Where("usuario_id = ?", usuarioID).Find(&coleccion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener la colección"})
		return
	}

	c.JSON(http.StatusOK, coleccion)
}

// Agregar una carta a la colección
func AgregarAColeccion(c *gin.Context) {
	var req AgregarCartaRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos", "detalle": err.Error()})
		return
	}

	nuevaCarta := models.ColeccionUsuario{
		UsuarioID:  req.UsuarioID,
		CartaApiID: req.CartaApiID,
		Cantidad:   req.Cantidad,
		EsFoil:     req.EsFoil,
	}

	if err := config.DB.Create(&nuevaCarta).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo agregar a la colección"})
		return
	}

	c.JSON(http.StatusCreated, nuevaCarta)
}
