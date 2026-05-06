package controllers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

func UploadImage(c *gin.Context) {
	// Obtener archivo del form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'image' no encontrado"})
		return
	}
	defer file.Close()

	// Validar extensión
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{
		".jpg": true, ".jpeg": true,
		".png": true, ".webp": true,
	}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tipo de archivo no permitido"})
		return
	}

	// Inicializar Firebase
	ctx := context.Background()
	opt := option.WithCredentialsFile("config/firebase_credentials.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al conectar con Firebase"})
		return
	}

	// Obtener cliente de Storage
	client, err := app.Storage(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al iniciar Storage"})
		return
	}

	// Obtener el bucket
	bucket, err := client.DefaultBucket()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al obtener bucket"})
		return
	}

	// Nombre único para el archivo
	filename := fmt.Sprintf("uploads/%d%s", time.Now().UnixNano(), ext)

	// Crear el objeto en Firebase Storage
	obj := bucket.Object(filename)
	writer := obj.NewWriter(ctx)
	writer.ContentType = "image/" + strings.TrimPrefix(ext, ".")

	// Copiar el archivo
	if _, err := io.Copy(writer, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al subir archivo"})
		return
	}

	// Cerrar el writer — importante, aquí es cuando se sube realmente
	if err := writer.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al finalizar subida"})
		return
	}

	// Hacer el archivo público
	if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al hacer público el archivo"})
		return
	}

	// Construir URL pública
	url := fmt.Sprintf(
		"https://storage.googleapis.com/proyectocartasbackfront-f5865.firebasestorage.app/%s",
		filename,
	)

	c.JSON(http.StatusOK, gin.H{"url": url})
}
