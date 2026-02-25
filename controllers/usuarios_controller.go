package controllers

import (
	"net/http"

	"ProyectoGinBack/config"
	"ProyectoGinBack/models"
	"ProyectoGinBack/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func CrearUsuario(c *gin.Context) {
	var usuario models.Usuario

	if err := c.ShouldBindJSON(&usuario); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datos inválidos",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(usuario.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al procesar el password",
		})
		return
	}

	usuario.Password = string(hashedPassword)

	config.DB.Create(&usuario)
	config.DB.First(&usuario, usuario.ID) //verificar si funciona el "refresh" con esto
	c.JSON(http.StatusCreated, usuario)
}

func ObtenerUsuarios(c *gin.Context) {
	var usuarios []models.Usuario
	config.DB.Find(&usuarios)

	c.JSON(http.StatusOK, usuarios)
}

func Login(c *gin.Context) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datos inválidos",
		})
		return
	}

	var usuario models.Usuario

	// Buscar usuario por email
	if err := config.DB.Where("email = ?", request.Email).First(&usuario).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Credenciales incorrectas",
		})
		return
	}

	// Comparar password plano vs hash
	err := bcrypt.CompareHashAndPassword(
		[]byte(usuario.Password),
		[]byte(request.Password),
	)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Credenciales incorrectas",
		})
		return
	}

	// Login exitoso (sin JWT aún)
	token, err := utils.GenerarToken(usuario.ID, usuario.RolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al generar el token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}
