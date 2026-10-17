package handlers

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"automarket/internal/auth"
)

const (
	MaxImageSize = 5 * 1024 * 1024 // 5 MB
)

var allowedImageExtensions = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
}

// SaveUploadedImage valida y almacena una imagen de vehículo en destDir.
// Retorna la URL pública relativa (ej. /uploads/car-xxx.jpg) o un error de validación.
func SaveUploadedImage(fileHeader *multipart.FileHeader, destDir string) (string, error) {
	if fileHeader == nil {
		return "", nil
	}

	if fileHeader.Size > MaxImageSize {
		return "", errors.New("la imagen supera el límite de 5MB")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	expectedMime, ok := allowedImageExtensions[ext]
	if !ok {
		return "", errors.New("formato de imagen no permitido (solo JPG, PNG o WEBP)")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", errors.New("error al leer el archivo de imagen")
	}
	defer src.Close()

	buf := make([]byte, 512)
	n, err := src.Read(buf)
	if err != nil && err != io.EOF {
		return "", errors.New("error al inspeccionar el contenido de la imagen")
	}

	detectedMime := http.DetectContentType(buf[:n])
	// Validar que el tipo detectado sea una imagen coherente con la extensión
	if detectedMime != expectedMime && !strings.HasPrefix(detectedMime, "image/") {
		return "", errors.New("el contenido no corresponde a una imagen válida")
	}

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", errors.New("error al procesar el archivo")
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("error al preparar directorio de almacenamiento: %w", err)
	}

	filename := fmt.Sprintf("car-%s%s", auth.GenerateToken()[:16], ext)
	targetPath := filepath.Join(destDir, filename)

	dst, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("error al guardar la imagen: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(targetPath)
		return "", fmt.Errorf("error al escribir la imagen: %w", err)
	}

	return "/uploads/" + filename, nil
}

// DeleteUploadedImage elimina de forma segura un archivo local previamente subido.
func DeleteUploadedImage(imageURL string, destDir string) {
	if !strings.HasPrefix(imageURL, "/uploads/") {
		return
	}
	filename := filepath.Base(imageURL)
	if filename == "" || filename == "." || filename == ".." {
		return
	}
	_ = os.Remove(filepath.Join(destDir, filename))
}
