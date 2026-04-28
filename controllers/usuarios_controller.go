package controllers

import (
	"net/http"

	"ProyectoGinBack/config"
	"ProyectoGinBack/dto"
	"ProyectoGinBack/models"
	"ProyectoGinBack/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UpdateUsuarioRequest struct {
	NombreUsuario string `json:"nombre_usuario"`
	Email         string `json:"email"`
	FotoPerfil    string `json:"foto_perfil"`
}

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

	config.DB.
		Preload("Rol").
		Find(&usuarios)

	var response []dto.UsuarioResponse
	for _, u := range usuarios {
		response = append(response, dto.MapUsuarioToDTO(u))
	}

	c.JSON(http.StatusOK, response)
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

func ActualizarUsuario(c *gin.Context) {
	userID := c.GetUint("user_id")

	var usuario models.Usuario
	if err := config.DB.First(&usuario, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	var request UpdateUsuarioRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// usuario.NombreUsuario = request.NombreUsuario
	// usuario.Email = request.Email
	if request.NombreUsuario != "" {
		usuario.NombreUsuario = request.NombreUsuario
	}

	if request.Email != "" {
		usuario.Email = request.Email
	}

	if request.FotoPerfil != "" {
		usuario.FotoPerfil = request.FotoPerfil
	}

	config.DB.Save(&usuario)

	c.JSON(http.StatusOK, gin.H{"message": "Usuario actualizado"})
}

func EliminarUsuario(c *gin.Context) {
	userID := c.GetUint("user_id")

	var usuario models.Usuario
	if err := config.DB.First(&usuario, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	config.DB.Delete(&usuario)

	c.JSON(http.StatusOK, gin.H{
		"message": "Usuario eliminado correctamente",
	})
}

func Register(c *gin.Context) {

	var request dto.RegisterRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datos inválidos",
		})
		return
	}

	// default de foto
	foto := request.FotoPerfil
	if foto == "" {
		foto = "https://i.pravatar.cc/150?img=1"
	}

	// verificar si email ya existe
	var existingUser models.Usuario
	if err := config.DB.Where("email = ?", request.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "El email ya está registrado",
		})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al verificar el usuario",
		})
		return
	}

	// hash del password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al procesar el password",
		})
		return
	}

	usuario := models.Usuario{
		NombreUsuario: request.NombreUsuario,
		Email:         request.Email,
		Password:      string(hashedPassword),
		RolID:         1,
		FotoPerfil:    foto,
	}

	if err := config.DB.Create(&usuario).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al crear el usuario",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Usuario registrado correctamente",
	})
}

func GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var usuario models.Usuario
	if err := config.DB.
		Preload("Rol").
		First(&usuario, userID).Error; err != nil {

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Usuario no encontrado",
		})
		return
	}

	// Opcional: usar DTO si no quieres exponer password
	response := dto.MapUsuarioToDTO(usuario)

	c.JSON(http.StatusOK, response)
}
