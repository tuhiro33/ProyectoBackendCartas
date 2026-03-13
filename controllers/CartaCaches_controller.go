package controllers

import (
	"ProyectoGinBack/config"
	"ProyectoGinBack/dto"
	"ProyectoGinBack/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SincronizarCarta guarda o actualiza una carta en el cache
func SincronizarCarta(c *gin.Context) {
	var input dto.CartaCacheDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	carta := models.CartaCache{
		ApiID:  input.ApiID,
		Juego:  input.Juego,
		Nombre: input.Nombre,
	}

	// "Upsert": Si existe la actualiza, si no, la crea
	result := config.DB.Where(models.CartaCache{ApiID: input.ApiID}).
		Assign(carta).
		FirstOrCreate(&carta)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al cachear carta"})
		return
	}

	c.JSON(http.StatusOK, carta)
}
