package dto

type CrearTransaccionDTO struct {
	PublicacionID uint    `json:"publicacion_id" binding:"required"`
	CompradorID   uint    `json:"comprador_id" binding:"required"`
	PrecioFinal   float64 `json:"precio_final" binding:"required"`
	EstadoPago    string  `json:"estado_pago" binding:"required"`
}
