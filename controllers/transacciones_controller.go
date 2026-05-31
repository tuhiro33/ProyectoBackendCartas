// Controlador de transacciones: registra y consulta las compras realizadas
// entre usuarios en el marketplace.
//
// Una transacción representa el momento en que un comprador adquiere
// una carta publicada por un vendedor. Al crearse, cambia automáticamente
// el estado de la publicación asociada a "vendido".
//
// Rutas que maneja (ver main.go):
//
//	POST /transacciones                      → CrearTransaccion       (protegida)
//	GET  /transacciones/historial/:usuarioId → ObtenerHistorialCompras (protegida)
package controllers

import (
	"ProyectoGinBack/config"
	"ProyectoGinBack/dto"
	"ProyectoGinBack/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CrearTransaccion registra una nueva compra y marca la publicación como vendida.
//
// POST /transacciones (protegida con JWT)
//
// ⚠️  PROBLEMAS IMPORTANTES:
//
//  1. SEGURIDAD — CompradorID viene del body del cliente, no del JWT.
//     Un usuario podría registrar compras a nombre de otro enviando
//     un CompradorID diferente al suyo.
//     Corrección: ignorar input.CompradorID y usar c.GetUint("user_id").
//
//  2. ATOMICIDAD — la transacción se guarda y el estado de la publicación
//     se actualiza en dos operaciones separadas sin una transacción de BD.
//     Si la segunda falla, la transacción queda registrada pero la publicación
//     sigue "Activa", creando una inconsistencia.
//     Corrección: envolver ambas operaciones en config.DB.Transaction()
//     igual que en MarcarComoVendida.
//
//  3. CONSISTENCIA — "vendido" (minúscula) vs "Vendida" (mayúscula) usado
//     en MarcarComoVendida y EliminarPublicacion. El valor del estado no está
//     centralizado como constante, lo que puede causar filtros fallidos.
//     Corrección: definir constantes en un paquete utils o models:
//     const EstadoActiva   = "Activa"
//     const EstadoVendida  = "Vendida"
//     const EstadoEliminada = "Eliminada"
//
//  4. SIN VALIDACIONES DE NEGOCIO — no verifica que:
//     - La publicación exista y esté "Activa" antes de comprarla.
//     - El comprador no sea el mismo vendedor (autocompra).
//     - El PrecioFinal coincida con el precio de la publicación.
func CrearTransaccion(c *gin.Context) {
	var input dto.CrearTransaccionDTO

	if err := c.ShouldBindJSON(&input); err != nil {
		// ⚠️  err.Error() expone detalles de validación internos al cliente.
		// Puede revelar nombres de campos o estructura del DTO.
		// En producción usar un mensaje genérico: gin.H{"error": "Datos inválidos"}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mapear DTO al modelo de BD.
	// CompradorID debería venir de c.GetUint("user_id") — ver advertencia arriba.
	transaccion := models.Transaccion{
		PublicacionID: input.PublicacionID,
		CompradorID:   input.CompradorID, // ⚠️  Usar c.GetUint("user_id") en su lugar
		PrecioFinal:   input.PrecioFinal,
		EstadoPago:    input.EstadoPago,
	}

	if err := config.DB.Create(&transaccion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo registrar la transacción"})
		return
	}

	// Actualizar estado de la publicación a "vendido" tras registrar la transacción.
	// ⚠️  Esta operación está fuera de la transacción de BD — si falla,
	// la transacción ya fue guardada pero la publicación queda "Activa".
	// ⚠️  "vendido" (minúscula) es inconsistente con "Vendida" usado en
	// MarcarComoVendida — unificar con una constante compartida.
	config.DB.Model(&models.PublicacionVenta{}).
		Where("id = ?", input.PublicacionID).
		Update("estado_publicacion", "vendido")

	// Devuelve el modelo directamente — si Transaccion tiene relaciones
	// (Publicacion, Comprador) no estarán cargadas en esta respuesta.
	c.JSON(http.StatusCreated, transaccion)
}

// ObtenerHistorialCompras devuelve todas las transacciones donde
// el usuario fue el comprador, con los datos de la publicación asociada.
//
// GET /transacciones/historial/:usuarioId (protegida con JWT)
//
// ⚠️  CONSIDERACIÓN DE SEGURIDAD: cualquier usuario autenticado puede ver
// el historial de compras de otro usuario conociendo su ID en la URL.
// Si el historial debe ser privado, verificar que coincida con el JWT:
//
//	userID := c.GetUint("user_id")
//	if fmt.Sprint(userID) != usuarioID { → 403 Forbidden }
//
// ⚠️  Preload("Publicacion") carga la publicación pero no sus relaciones
// anidadas (Vendedor, Coleccion.Carta). El historial mostrará datos
// limitados de cada compra. Para un historial completo considerar:
//
//	Preload("Publicacion.Vendedor")
//	Preload("Publicacion.Coleccion.Carta")
func ObtenerHistorialCompras(c *gin.Context) {
	usuarioID := c.Param("usuarioId")

	var transacciones []models.Transaccion

	result := config.DB.
		Preload("Publicacion"). // Carga datos de la publicación comprada
		Where("comprador_id = ?", usuarioID).
		Find(&transacciones)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener historial"})
		return
	}

	// Devuelve el modelo directamente sin DTO.
	// Si Transaccion incluye campos sensibles considerar un DTO de respuesta.
	c.JSON(http.StatusOK, transacciones)
}
