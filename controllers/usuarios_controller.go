// Controlador de usuarios: maneja el ciclo de vida de cuentas de usuario.
// Implementa la capa C del patrón MVC — recibe peticiones HTTP de las rutas
// definidas en main.go, interactúa con la BD via config.DB (GORM), y devuelve
// respuestas JSON al cliente.
//
// Rutas que maneja (ver main.go):
//
//	POST   /usuarios          → CrearUsuario   (público, sin JWT)
//	POST   /register          → Register       (público, con validaciones extra)
//	POST   /login             → Login          (público, devuelve JWT)
//	GET    /me                → GetProfile     (protegida)
//	GET    /usuarios          → ObtenerUsuarios (protegida)
//	PUT    /usuarios          → ActualizarUsuario (protegida)
//	DELETE /usuarios          → EliminarUsuario   (protegida)
//	GET    /usuarios/perfil/:usuarioId → ObtenerPerfilPublico (pública)
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

// UpdateUsuarioRequest define los campos que el usuario puede modificar en su perfil.
// Todos son opcionales — solo se actualiza lo que venga con valor no vacío.
// Se define aquí y no en /dto porque solo se usa internamente en este controlador.
//
// ⚠️  No incluye RolID intencionalmente: un usuario no puede cambiar su propio rol.
// El cambio de roles debería ser una operación exclusiva del administrador.
type UpdateUsuarioRequest struct {
	NombreUsuario string `json:"nombre_usuario"`
	Email         string `json:"email"`
	FotoPerfil    string `json:"foto_perfil"`
	Password      string `json:"password"`
}

// CrearUsuario crea un nuevo usuario directamente desde el modelo, sin validaciones extra.
// A diferencia de Register, no verifica duplicados de email ni asigna rol por defecto.
//
// POST /usuarios (ruta pública, sin JWT)
//
// ⚠️  PROBLEMAS CONOCIDOS:
//  1. Recibe models.Usuario directamente en el bind, lo que expone todos los campos
//     del modelo al cliente — incluyendo RolID, que podría enviarse para auto-asignarse
//     rol de administrador. Debería usarse un DTO de entrada (RegisterRequest).
//  2. config.DB.First(&usuario, usuario.ID) después del Create intenta un "refresh"
//     pero no hace Preload("Rol"), por lo que usuario.Rol quedará vacío en la respuesta.
//  3. La respuesta devuelve models.Usuario directamente, exponiendo el hash del Password.
//     Debería usar dto.MapUsuarioToDTO(usuario).
//  4. No verifica si el email ya existe (a diferencia de Register).
//
// Esta ruta parece ser un endpoint legado — Register es la versión más completa y segura.
// Considerar deprecarla o unificarla con Register.
func CrearUsuario(c *gin.Context) {
	var usuario models.Usuario

	// ShouldBindJSON deserializa el body JSON en la struct.
	// Si falta un campo con binding:"required" o el JSON es inválido, retorna error.
	if err := c.ShouldBindJSON(&usuario); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datos inválidos",
		})
		return
	}

	// Hashear la contraseña antes de guardarla.
	// bcrypt.DefaultCost = 10 iteraciones — balance entre seguridad y rendimiento.
	// NUNCA guardar contraseñas en texto plano.
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
	// Intenta refrescar el objeto desde la BD para obtener FechaRegistro y otros
	// campos autogenerados. Sin Preload("Rol") el campo Rol quedará vacío.
	config.DB.First(&usuario, usuario.ID)

	// ❌ Expone el hash de Password en la respuesta.
	// ✅ Corrección: c.JSON(http.StatusCreated, dto.MapUsuarioToDTO(usuario))
	c.JSON(http.StatusCreated, usuario)
}

// ObtenerUsuarios devuelve la lista completa de usuarios registrados.
// Usa Preload("Rol") para cargar el nombre del rol de cada usuario,
// y MapUsuarioToDTO para excluir el hash de contraseña de la respuesta.
//
// GET /usuarios (protegida con JWT)
//
// ⚠️  Sin paginación: si hay muchos usuarios, esta consulta carga
// todos en memoria. Considerar agregar limit/offset o paginación con cursor.
func ObtenerUsuarios(c *gin.Context) {
	var usuarios []models.Usuario

	config.DB.
		Preload("Rol"). // Carga el objeto Rol asociado para que MapUsuarioToDTO pueda leer Rol.Nombre
		Find(&usuarios)

	// Convertir cada usuario al DTO de respuesta (excluye Password, aplana Rol)
	var response []dto.UsuarioResponse
	for _, u := range usuarios {
		response = append(response, dto.MapUsuarioToDTO(u))
	}

	// Nota: si no hay usuarios, response será nil y se serializa como null en JSON.
	// Para devolver [] en lugar de null cuando está vacío:
	//   response := make([]dto.UsuarioResponse, 0)
	c.JSON(http.StatusOK, response)
}

// Login autentica a un usuario con email y contraseña, y devuelve un token JWT.
// El token debe enviarse en peticiones posteriores como: Authorization: Bearer <token>
//
// POST /login (ruta pública)
//
// Seguridad: ambos casos de error (usuario no encontrado y contraseña incorrecta)
// devuelven el mismo mensaje "Credenciales incorrectas" — esto es intencional
// para evitar revelar si un email está registrado (user enumeration attack).
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

	// Buscar usuario por email. Si no existe, responder con 401 genérico.
	if err := config.DB.Where("email = ?", request.Email).First(&usuario).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Credenciales incorrectas", // Mismo mensaje que contraseña incorrecta
		})
		return
	}

	// CompareHashAndPassword verifica que request.Password coincida con el hash
	// almacenado. bcrypt incluye el salt en el hash, por lo que la comparación
	// es segura contra ataques de rainbow table.
	err := bcrypt.CompareHashAndPassword(
		[]byte(usuario.Password),
		[]byte(request.Password),
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Credenciales incorrectas", // Mismo mensaje deliberadamente
		})
		return
	}

	// Credenciales válidas — generar JWT con UserID y RolID embebidos.
	// El comentario "sin JWT aún" en el código original es un residuo
	// de una versión anterior; el JWT ya está implementado aquí.
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

// ActualizarUsuario modifica los datos del perfil del usuario autenticado.
// Solo actualiza los campos que vengan con valor no vacío (actualización parcial).
// Un usuario solo puede modificar su propio perfil — el ID viene del JWT, no de la URL.
//
// PUT /usuarios (protegida con JWT)
func ActualizarUsuario(c *gin.Context) {
	// Obtener el ID del usuario desde el JWT (inyectado por AuthMiddleware).
	// Esto garantiza que cada usuario solo pueda editar su propio perfil.
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

	// Actualización parcial: solo modifica los campos que vengan con valor.
	// El código comentado arriba habría sobreescrito con cadena vacía
	// si el cliente no enviaba el campo — este enfoque es más seguro.
	if request.NombreUsuario != "" {
		usuario.NombreUsuario = request.NombreUsuario
	}
	if request.Email != "" {
		// ⚠️  No verifica si el nuevo email ya está en uso por otro usuario.
		// Podría causar un error de constraint unique en la BD sin un mensaje claro.
		// Recomendación: verificar con DB.Where("email = ? AND id != ?", email, userID)
		usuario.Email = request.Email
	}
	if request.FotoPerfil != "" {
		usuario.FotoPerfil = request.FotoPerfil
	}
	if request.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword(
			[]byte(request.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar password"})
			return
		}
		usuario.Password = string(hashedPassword)
	}

	// Save actualiza TODOS los campos del struct en la BD (UPDATE completo),
	// no solo los modificados. Es seguro aquí porque primero se hizo First()
	// para cargar los valores actuales antes de modificar.
	config.DB.Save(&usuario)

	c.JSON(http.StatusOK, gin.H{"message": "Usuario actualizado"})
}

// EliminarUsuario elimina permanentemente la cuenta del usuario autenticado.
// Opera sobre el usuario del JWT — nadie puede eliminar la cuenta de otro.
//
// DELETE /usuarios (protegida con JWT)
//
// ⚠️  Eliminación física (hard delete): el registro desaparece de la BD.
// Si se necesita auditoría o recuperación de cuenta, considerar soft delete
// agregando gorm.DeletedAt al modelo (GORM lo gestiona automáticamente).
func EliminarUsuario(c *gin.Context) {
	userID := c.GetUint("user_id")

	var usuario models.Usuario
	if err := config.DB.First(&usuario, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	// ⚠️  No verifica registros relacionados antes de eliminar.
	// Dependiendo de las restricciones FK en la BD, esto podría fallar
	// si el usuario tiene colecciones, publicaciones o transacciones activas.
	config.DB.Delete(&usuario)

	c.JSON(http.StatusOK, gin.H{
		"message": "Usuario eliminado correctamente",
	})
}

// Register crea una nueva cuenta con validaciones completas.
// Es la versión robusta de CrearUsuario — usa DTO, verifica duplicados,
// asigna rol por defecto y no expone el modelo interno directamente.
//
// POST /register (ruta pública)
func Register(c *gin.Context) {
	var request dto.RegisterRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datos inválidos",
		})
		return
	}

	// Foto de perfil por defecto si el cliente no envía una.
	// pravatar.cc es un servicio de avatares placeholder para desarrollo.
	// En producción considerar una imagen propia hosteada en el servidor.
	foto := request.FotoPerfil
	if foto == "" {
		foto = "https://i.pravatar.cc/150?img=1"
	}

	// Verificar unicidad del email antes de intentar insertar.
	// Si err == nil significa que SÍ encontró un usuario → email duplicado.
	// Si err == ErrRecordNotFound → email disponible, continuar.
	// Cualquier otro error → problema de BD.
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
		RolID:         1, // Rol por defecto: usuario regular (ID=1)
		FotoPerfil:    foto,
	}

	if err := config.DB.Create(&usuario).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al crear el usuario",
		})
		return
	}

	// Responde solo con mensaje de éxito — no devuelve el objeto usuario
	// para evitar exponer datos en el momento del registro.
	c.JSON(http.StatusCreated, gin.H{
		"message": "Usuario registrado correctamente",
	})
}

// GetProfile devuelve el perfil completo del usuario autenticado.
// Usa Preload("Rol") y MapUsuarioToDTO para una respuesta segura y completa.
//
// GET /me (protegida con JWT)
func GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var usuario models.Usuario
	if err := config.DB.
		Preload("Rol"). // Necesario para que MapUsuarioToDTO pueda leer Rol.Nombre
		First(&usuario, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Usuario no encontrado",
		})
		return
	}

	c.JSON(http.StatusOK, dto.MapUsuarioToDTO(usuario))
}

// ObtenerPerfilPublico devuelve los datos básicos de cualquier usuario por su ID.
// Usa el mismo DTO que GetProfile, que ya excluye Password y aplana el Rol.
//
// GET /usuarios/perfil/:usuarioId (ruta pública, sin JWT)
//
// ⚠️  Devuelve Email y FechaRegistro que podrían considerarse datos privados.
// Para un perfil verdaderamente público considerar un DTO más restrictivo
// que solo exponga NombreUsuario y FotoPerfil.
func ObtenerPerfilPublico(c *gin.Context) {
	usuarioID := c.Param("usuarioId")

	var usuario models.Usuario
	if err := config.DB.Preload("Rol").First(&usuario, usuarioID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "El usuario solicitado no existe",
		})
		return
	}

	c.JSON(http.StatusOK, dto.MapUsuarioToDTO(usuario))
}
