package controllers

import (
	"net/http"
	"strconv"

	"ProyectoGinBack/config"
	"ProyectoGinBack/models"

	"github.com/gin-gonic/gin"
)

// Listar roles (público)
func ObtenerRoles(c *gin.Context) {
	var roles []models.Rol
	if err := config.DB.Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener roles"})
		return
	}
	c.JSON(http.StatusOK, roles)
}

// Crear rol (recomendado: admin only)
func CrearRol(c *gin.Context) {
	var rol models.Rol
	if err := c.ShouldBindJSON(&rol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// evitar duplicados por nombre
	var existing models.Rol
	if err := config.DB.Where("nombre = ?", rol.Nombre).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El rol ya existe"})
		return
	}

	if err := config.DB.Create(&rol).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el rol"})
		return
	}

	c.JSON(http.StatusCreated, rol)
}

// Actualizar rol (admin only)
func ActualizarRol(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	var rol models.Rol
	if err := config.DB.First(&rol, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rol no encontrado"})
		return
	}

	var body models.Rol
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	rol.Nombre = body.Nombre

	if err := config.DB.Save(&rol).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar el rol"})
		return
	}

	c.JSON(http.StatusOK, rol)
}

// Eliminar rol (admin only)
func EliminarRol(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	var rol models.Rol
	if err := config.DB.First(&rol, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rol no encontrado"})
		return
	}

	// OJO: si hay usuarios ligados, podrías bloquear el delete o reasignar
	if err := config.DB.Delete(&rol).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar el rol"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rol eliminado"})
}
