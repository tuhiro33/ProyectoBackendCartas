package controllers

import (
	"net/http"

	"ProyectoGinBack/config"
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

func ObtenerPublicaciones(c *gin.Context) {
	var publicaciones []models.PublicacionVenta

	result := config.DB.Find(&publicaciones)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener publicaciones",
		})
		return
	}

	c.JSON(http.StatusOK, publicaciones)
}

func ObtenerPublicacionPorID(c *gin.Context) {
	id := c.Param("id")

	var publicacion models.PublicacionVenta

	result := config.DB.First(&publicacion, id)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Publicación no encontrada",
		})
		return
	}

	c.JSON(http.StatusOK, publicacion)
}

func CrearPublicacion(c *gin.Context) {
	var req CrearPublicacionRequest

	// 1️⃣ Leer JSON del body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos inválidos",
			"detalle": err.Error(),
		})
		return
	}

	// 2️⃣ Crear modelo
	publicacion := models.PublicacionVenta{
		VendedorID:        req.VendedorID,
		ColeccionID:       req.ColeccionID,
		Precio:            req.Precio,
		EstadoCarta:       req.EstadoCarta,
		FotoURL:           req.FotoURL,
		EstadoPublicacion: req.EstadoPublicacion,
	}

	// 3️⃣ Guardar en DB
	if err := config.DB.Create(&publicacion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "No se pudo crear la publicación",
		})
		return
	}

	// 4️⃣ Responder
	c.JSON(http.StatusCreated, publicacion)
}
