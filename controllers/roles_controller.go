// Controlador de roles: gestiona el catálogo de roles del sistema.
// Los roles definen los permisos de cada usuario (ej: rol 1 = usuario, rol 2 = admin).
//
// Rutas que maneja (ver main.go):
//
//	GET    /roles          → ObtenerRoles   (pública)
//	POST   /admin/roles    → CrearRol       (admin only: JWT + RequireRoles(2))
//	PUT    /admin/roles/:id → ActualizarRol (admin only: JWT + RequireRoles(2))
//	DELETE /admin/roles/:id → EliminarRol   (admin only: JWT + RequireRoles(2))
//
// La protección de las rutas admin está en main.go mediante la cadena:
//
//	AuthMiddleware() → RequireRoles(2)
//
// Este controlador no necesita verificar roles internamente.
package controllers

import (
	"net/http"
	"strconv"

	"ProyectoGinBack/config"
	"ProyectoGinBack/models"

	"github.com/gin-gonic/gin"
)

// ObtenerRoles devuelve todos los roles disponibles en el sistema.
// Es una ruta pública — el frontend la usa para poblar selectores
// o mostrar el nombre del rol sin necesidad de autenticación.
//
// GET /roles (pública)
func ObtenerRoles(c *gin.Context) {
	var roles []models.Rol

	// Find sin condiciones trae todos los registros de la tabla 'roles'.
	// En un sistema típico hay pocos roles (2-5), por lo que no se necesita paginación.
	if err := config.DB.Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener roles"})
		return
	}

	c.JSON(http.StatusOK, roles)
}

// CrearRol agrega un nuevo rol al sistema.
// Verifica que el nombre no esté duplicado antes de insertar.
//
// POST /admin/roles (admin only)
//
// ⚠️  Recibe models.Rol directamente en el bind en lugar de un DTO.
// Esto expone todos los campos del modelo al cliente — si Rol tiene
// campos sensibles o autogenerados (como ID), el cliente podría
// intentar fijarlos. Para este modelo simple es aceptable,
// pero un DTO de entrada sería más robusto.
func CrearRol(c *gin.Context) {
	var rol models.Rol

	// ShouldBindJSON deserializa el body JSON en la struct Rol.
	if err := c.ShouldBindJSON(&rol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// Verificar unicidad del nombre antes de insertar.
	// Si err == nil significa que SÍ encontró un rol con ese nombre → duplicado.
	// Mismo patrón que la verificación de email en Register.
	var existing models.Rol
	if err := config.DB.Where("nombre = ?", rol.Nombre).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El rol ya existe"})
		return
	}
	// Si err != nil y es ErrRecordNotFound → nombre disponible, continuar.
	// Otros errores de BD se ignoran silenciosamente aquí — podría ser
	// más robusto verificar que el error sea específicamente ErrRecordNotFound.

	if err := config.DB.Create(&rol).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el rol"})
		return
	}

	c.JSON(http.StatusCreated, rol)
}

// ActualizarRol modifica el nombre de un rol existente por su ID.
//
// PUT /admin/roles/:id (admin only)
//
// ⚠️  strconv.Atoi convierte el parámetro de URL (siempre string) a entero.
// El segundo valor de retorno es el error de conversión — aquí se ignora con _.
// Si :id no es un número válido (ej: "/admin/roles/abc"), Atoi devuelve 0
// y config.DB.First(&rol, 0) no encontrará ningún registro, respondiendo
// con 404. Es un manejo implícito que funciona pero no da un mensaje claro.
// Más explícito:
//
//	id, err := strconv.Atoi(idStr)
//	if err != nil {
//	    c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
//	    return
//	}
func ActualizarRol(c *gin.Context) {
	idStr := c.Param("id")       // Lee el parámetro :id de la URL como string
	id, _ := strconv.Atoi(idStr) // Convierte a int — error ignorado, ver advertencia

	// Buscar el rol existente antes de modificarlo.
	// Si no existe, responder con 404 antes de procesar el body.
	var rol models.Rol
	if err := config.DB.First(&rol, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rol no encontrado"})
		return
	}

	// body recibe solo los campos nuevos del cliente.
	// Se usa una variable separada para no sobreescribir
	// el ID y otros campos del rol cargado desde la BD.
	var body models.Rol
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// Solo se actualiza el nombre — otros campos del rol no son modificables.
	// ⚠️  No verifica si el nuevo nombre ya está en uso por otro rol,
	// lo que podría causar un error de constraint unique en la BD
	// sin un mensaje claro al cliente.
	rol.Nombre = body.Nombre

	// Save hace UPDATE completo del registro con todos sus campos.
	// Es seguro aquí porque se hizo First() antes para cargar
	// los valores actuales — no se pierden datos no enviados.
	if err := config.DB.Save(&rol).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar el rol"})
		return
	}

	c.JSON(http.StatusOK, rol)
}

// EliminarRol elimina un rol del sistema por su ID.
//
// DELETE /admin/roles/:id (admin only)
//
// ⚠️  RIESGO DE INTEGRIDAD REFERENCIAL:
// Si hay usuarios con este rol asignado (usuarios.rol_id = id),
// eliminar el rol puede causar:
//   - Error de FK constraint si la BD tiene restricciones estrictas
//   - Usuarios con rol_id huérfano si la BD no tiene restricciones
//
// El comentario en el código original señala este riesgo correctamente.
// Opciones para manejarlo:
//  1. Verificar antes de eliminar:
//     var count int64
//     config.DB.Model(&models.Usuario{}).Where("rol_id = ?", id).Count(&count)
//     if count > 0 → responder con error descriptivo
//  2. Reasignar usuarios al rol por defecto (ID=1) antes de eliminar.
//  3. No permitir eliminar roles base (ID=1 e ID=2).
func EliminarRol(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr) // Error de conversión ignorado — mismo caso que ActualizarRol

	var rol models.Rol
	if err := config.DB.First(&rol, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rol no encontrado"})
		return
	}

	// Hard delete: elimina el registro físicamente de la tabla 'roles'.
	// Si models.Rol tuviera gorm.DeletedAt, este sería un soft delete automático.
	if err := config.DB.Delete(&rol).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar el rol"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rol eliminado"})
}
