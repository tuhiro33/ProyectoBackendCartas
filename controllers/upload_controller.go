// Controlador de subida de imágenes: gestiona la carga de archivos de imagen
// hacia Firebase Storage (Google Cloud Storage) y devuelve la URL pública.
//
// Es usado por dos páginas del frontend:
//   - Coleccion.tsx: subir foto real de una carta al publicarla en el mercado
//   - Perfil.tsx:    subir nueva foto de perfil del usuario
//
// Ruta que maneja (ver main.go):
//
//	POST /upload → UploadImage (pública — ver advertencia de seguridad abajo)
//
// ⚠️  SEGURIDAD: esta ruta está fuera del grupo auth en main.go,
// lo que significa que cualquier persona puede subir imágenes sin autenticarse.
// Considerar moverla dentro del grupo auth para evitar abuso del almacenamiento.
package controllers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

// UploadImage recibe una imagen del frontend, la valida, la sube a
// Firebase Storage y devuelve la URL pública para guardarla en la BD.
//
// El archivo debe enviarse como multipart/form-data con el campo "image"
// — mismo nombre que usa apiClient.post("/upload", formData) en el frontend.
//
// Respuesta exitosa:
//
//	{ "url": "https://storage.googleapis.com/proyectocartasbackfront-f5865.firebasestorage.app/uploads/..." }
//
// ⚠️  RENDIMIENTO: inicializa una nueva conexión a Firebase en CADA petición.
// firebase.NewApp() + app.Storage() + client.Bucket() son operaciones costosas
// que deberían ejecutarse una sola vez al arrancar el servidor (en config/)
// y reutilizarse aquí como una variable global, igual que config.DB con GORM.
func UploadImage(c *gin.Context) {

	// -----------------------------------------------------------------------
	// PASO 1: Extraer el archivo del formulario multipart
	// c.Request.FormFile("image") lee el campo "image" del body multipart.
	// Devuelve tres valores:
	//   file   → io.ReadCloser para leer el contenido binario del archivo
	//   header → metadatos del archivo (nombre original, tamaño, tipo MIME)
	//   err    → error si el campo no existe o el body no es multipart
	// defer file.Close() garantiza que el archivo se cierra al terminar
	// la función, liberando la memoria del buffer temporal.
	// -----------------------------------------------------------------------
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'image' no encontrado"})
		return
	}
	defer file.Close()

	// -----------------------------------------------------------------------
	// PASO 2: Validar la extensión del archivo
	// filepath.Ext extrae la extensión del nombre original del archivo:
	//   "carta.jpg" → ".jpg", "foto.PNG" → ".PNG"
	// strings.ToLower normaliza a minúsculas para que ".JPG" y ".jpg"
	// se traten igual.
	//
	// El mapa allowed actúa como un conjunto de extensiones permitidas —
	// allowed[ext] devuelve true si la extensión es válida, false si no existe.
	//
	// ⚠️  Validar solo la extensión no es suficiente para seguridad real —
	// un archivo malicioso puede tener extensión .jpg pero contenido peligroso.
	// En producción se recomienda validar también el Content-Type o los
	// primeros bytes del archivo (magic bytes) para verificar que realmente
	// es una imagen.
	// -----------------------------------------------------------------------
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{
		".jpg": true, ".jpeg": true,
		".png": true, ".webp": true,
	}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tipo de archivo no permitido"})
		return
	}

	// -----------------------------------------------------------------------
	// PASO 3: Inicializar Firebase con credenciales de variable de entorno
	//
	// El código comentado arriba usaba un archivo físico de credenciales
	// (config/Firebase_Credentials.json), lo que es problemático en
	// producción porque el archivo no se puede incluir en el repositorio.
	//
	// La solución actual lee las credenciales JSON desde la variable de
	// entorno Firebase_Credentials definida en Railway, y las pasa
	// directamente como []byte con option.WithCredentialsJSON.
	// Esto es más seguro y portable — funciona en cualquier entorno
	// sin archivos físicos.
	// -----------------------------------------------------------------------
	ctx := context.Background() // Contexto sin timeout — ver nota abajo

	firebaseJSON := os.Getenv("Firebase_Credentials")
	if firebaseJSON == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Variable de entorno 'Firebase_Credentials' no configurada"})
		return
	}

	// option.WithCredentialsJSON convierte el string JSON de la variable
	// de entorno a las credenciales que Firebase SDK necesita para autenticarse
	// con Google Cloud Storage.
	opt := option.WithCredentialsJSON([]byte(firebaseJSON))
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al conectar con Firebase"})
		return
	}

	// -----------------------------------------------------------------------
	// PASO 4: Obtener cliente y bucket de Firebase Storage
	// app.Storage() devuelve el cliente de Cloud Storage autenticado.
	// client.Bucket() apunta al bucket específico del proyecto.
	// El nombre del bucket está hardcodeado — considerar moverlo a
	// variable de entorno: os.Getenv("FIREBASE_BUCKET")
	// -----------------------------------------------------------------------
	client, err := app.Storage(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al iniciar Storage"})
		return
	}

	bucket, err := client.Bucket("proyectocartasbackfront-f5865.firebasestorage.app")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al obtener bucket"})
		return
	}

	// -----------------------------------------------------------------------
	// PASO 5: Generar nombre único para el archivo
	// time.Now().UnixNano() devuelve los nanosegundos desde epoch Unix —
	// un número de 19 dígitos único en cada llamada que evita colisiones
	// de nombres sin necesidad de UUIDs o bases de datos.
	// Ejemplo: "uploads/1710432891234567890.jpg"
	//
	// ⚠️  En casos extremos de concurrencia muy alta, dos peticiones
	// simultáneas podrían generar el mismo nanosegundo. Para mayor
	// robustez considerar: fmt.Sprintf("uploads/%d_%s%s", UnixNano, uuid, ext)
	// -----------------------------------------------------------------------
	filename := fmt.Sprintf("uploads/%d%s", time.Now().UnixNano(), ext)

	// -----------------------------------------------------------------------
	// PASO 6: Subir el archivo a Firebase Storage
	// bucket.Object(filename) crea la referencia al objeto (archivo) en el bucket.
	// obj.NewWriter(ctx) abre un stream de escritura hacia Firebase Storage.
	// writer.ContentType informa a los navegadores cómo tratar el archivo
	// al acceder a la URL pública — importante para que las imágenes
	// se muestren en lugar de descargarse.
	// strings.TrimPrefix(ext, ".") elimina el punto: ".jpg" → "jpg"
	// para construir "image/jpg".
	// -----------------------------------------------------------------------
	obj := bucket.Object(filename)
	writer := obj.NewWriter(ctx)
	writer.ContentType = "image/" + strings.TrimPrefix(ext, ".")

	// io.Copy lee el archivo del formulario y lo escribe al stream de Firebase.
	// No carga todo el archivo en memoria — lo transfiere en chunks,
	// lo que es eficiente para archivos grandes.
	if _, err := io.Copy(writer, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al subir archivo"})
		return
	}

	// writer.Close() finaliza y confirma la subida a Firebase Storage.
	// ⚠️  Este es el paso crítico — sin Close(), el archivo queda
	// en un estado incompleto y no será accesible en Storage.
	// Los errores de red o cuota se manifiestan aquí, no en io.Copy.
	if err := writer.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al finalizar subida"})
		return
	}

	// -----------------------------------------------------------------------
	// PASO 7: Hacer el archivo públicamente accesible
	// Por defecto los objetos en Firebase Storage son privados.
	// ACL (Access Control List) define quién puede acceder al archivo.
	// storage.AllUsers = cualquier persona en internet
	// storage.RoleReader = permiso de solo lectura (no puede modificar ni eliminar)
	// Esto es necesario para que las URLs de imágenes funcionen en el frontend.
	// -----------------------------------------------------------------------
	if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al hacer público el archivo"})
		return
	}

	// -----------------------------------------------------------------------
	// PASO 8: Construir y devolver la URL pública
	// La URL sigue el patrón estándar de Google Cloud Storage:
	//   https://storage.googleapis.com/{bucket}/{objeto}
	// Esta URL se guarda en la BD (foto_url en PublicacionVenta o
	// foto_perfil en Usuario) para mostrar la imagen en el frontend.
	// -----------------------------------------------------------------------
	url := fmt.Sprintf(
		"https://storage.googleapis.com/proyectocartasbackfront-f5865.firebasestorage.app/%s",
		filename,
	)

	c.JSON(http.StatusOK, gin.H{"url": url})
}
