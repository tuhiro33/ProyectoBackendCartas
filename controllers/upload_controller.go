package controllers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	supa "github.com/supabase-community/supabase-go"
)

func UploadImage(c *gin.Context) {
	// Obtener el archivo del form
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

	// Leer bytes
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al leer archivo"})
		return
	}

	// Nombre único
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	storagePath := "uploads/" + filename

	// Subir a Supabase Storage
	client, err := supa.NewClient(
		os.Getenv("SUPABASE_URL"),
		os.Getenv("SUPABASE_KEY"),
		nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al conectar con Supabase"})
		return
	}

	reader := bytes.NewReader(fileBytes)

	_, err = client.Storage.UploadFile("imagenes", storagePath, reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al subir imagen"})
		return
	}

	// URL pública
	url := fmt.Sprintf(
		"%s/storage/v1/object/public/imagenes/%s",
		os.Getenv("SUPABASE_URL"),
		storagePath,
	)

	c.JSON(http.StatusOK, gin.H{"url": url})
}
