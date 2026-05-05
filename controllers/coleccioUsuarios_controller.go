package controllers

import (
	"ProyectoGinBack/config"
	"ProyectoGinBack/dto"
	"ProyectoGinBack/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Obtener toda la colección de un usuario
func ObtenerColeccionUsuario(c *gin.Context) {
	usuarioID := c.Param("usuarioId")
	var coleccion []models.ColeccionUsuario

	if err := config.DB.Preload("Carta").Where("usuario_id = ?", usuarioID).Find(&coleccion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener la colección"})
		return
	}
	c.JSON(http.StatusOK, coleccion)
}

// Agregar una carta a la colección
func AgregarAColeccion(c *gin.Context) {
	var req dto.AgregarCartaRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// 1. CREACIÓN AUTOMÁTICA EN CARTA_CACHE
	// Usamos Save para que si el ApiID ya existe, solo actualice los datos (como la URL de imagen)
	// Si no existe, lo inserta automáticamente.
	cartaAuto := models.CartaCache{
		ApiID:     req.Carta.ApiID,
		Nombre:    req.Carta.Nombre,
		Juego:     req.Carta.Juego,
		UrlImagen: req.Carta.UrlImagen, // ← este campo
	}

	// config.DB.Save detecta la llave primaria (ApiID) y decide si hace INSERT o UPDATE
	if err := config.DB.Save(&cartaAuto).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear/actualizar la referencia de la carta"})
		return
	}

	// 2. AHORA SÍ, AGREGAR A LA COLECCIÓN
	// Como la línea de arriba ya aseguró que la carta existe, esta parte no fallará por FK
	nuevaEntrada := models.ColeccionUsuario{
		UsuarioID:  req.UsuarioID,
		CartaApiID: req.Carta.ApiID,
		Cantidad:   req.Cantidad,
		EsFoil:     req.EsFoil,
	}

	if err := config.DB.Create(&nuevaEntrada).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al vincular la carta al usuario"})
		return
	}

	c.JSON(http.StatusCreated, nuevaEntrada)
}

func EliminarDeColeccion(c *gin.Context) {
	// Obtenemos el ID desde la URL (ej: /coleccion/5)
	id := c.Param("id")

	// Ejecutamos el Delete en GORM
	// Usamos Unscoped() solo si tienes Soft Delete (DeletedAt) y quieres borrarlo de verdad,
	// si no, config.DB.Delete basta.
	result := config.DB.Delete(&models.ColeccionUsuario{}, id)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el registro"})
		return
	}

	// Verificamos si realmente se eliminó algo (si el ID existía)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontró la carta con ese ID"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Carta eliminada de la colección con éxito"})
}
