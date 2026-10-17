package handlers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"automarket/internal/auth"
	"automarket/internal/domain"
	"automarket/internal/handlers"
	"automarket/internal/store"
)

// Genera un archivo multipart en memoria
func createMultipartFileHeader(filename string, content []byte) (*multipart.FileHeader, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("photo", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(content); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "/", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if err := req.ParseMultipartForm(10 << 20); err != nil {
		return nil, err
	}
	return req.MultipartForm.File["photo"][0], nil
}

func TestSaveUploadedImage_ValidTypes(t *testing.T) {
	tempDir := t.TempDir()

	// Magic bytes mínimos para JPEG y PNG
	jpegBytes := []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x01\x00`\x00`\x00\x00\xff\xdb\x00C\x00")
	pngBytes := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00")

	// 1. Probar JPEG
	fh, err := createMultipartFileHeader("car.jpg", jpegBytes)
	if err != nil {
		t.Fatalf("error creating header: %v", err)
	}
	url, err := handlers.SaveUploadedImage(fh, tempDir)
	if err != nil {
		t.Fatalf("expected valid jpeg save, got err: %v", err)
	}
	if !strings.HasPrefix(url, "/uploads/car-") || !strings.HasSuffix(url, ".jpg") {
		t.Errorf("unexpected url format: %s", url)
	}

	// 2. Probar PNG
	fhPng, err := createMultipartFileHeader("auto.png", pngBytes)
	if err != nil {
		t.Fatalf("error creating header: %v", err)
	}
	urlPng, err := handlers.SaveUploadedImage(fhPng, tempDir)
	if err != nil {
		t.Fatalf("expected valid png save, got err: %v", err)
	}
	if !strings.HasSuffix(urlPng, ".png") {
		t.Errorf("unexpected png url: %s", urlPng)
	}

	// 3. Probar DeleteUploadedImage
	handlers.DeleteUploadedImage(url, tempDir)
	savedFile := filepath.Join(tempDir, filepath.Base(url))
	if _, err := os.Stat(savedFile); !os.IsNotExist(err) {
		t.Errorf("expected file %s to be deleted", savedFile)
	}
}

func TestSaveUploadedImage_Rejections(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Extensión no permitida
	fh, _ := createMultipartFileHeader("script.sh", []byte("#!/bin/bash\necho hack"))
	_, err := handlers.SaveUploadedImage(fh, tempDir)
	if err == nil {
		t.Error("expected error on .sh file, got nil")
	}

	// 2. Extensión .jpg pero contenido texto (spoofing)
	fhFake, _ := createMultipartFileHeader("fake.jpg", []byte("plain text content pretending to be image"))
	_, err = handlers.SaveUploadedImage(fhFake, tempDir)
	if err == nil {
		t.Error("expected error on spoofed text file, got nil")
	}

	// 3. Archivo nil retorna vacío sin error
	emptyURL, err := handlers.SaveUploadedImage(nil, tempDir)
	if err != nil || emptyURL != "" {
		t.Errorf("expected empty string and nil error on nil fileHeader, got: %s, %v", emptyURL, err)
	}

	// 4. Archivo mayor a 5MB
	oversized := make([]byte, handlers.MaxImageSize+1024)
	fhLarge, _ := createMultipartFileHeader("large.png", oversized)
	_, err = handlers.SaveUploadedImage(fhLarge, tempDir)
	if err == nil || !strings.Contains(err.Error(), "5MB") {
		t.Errorf("expected 5MB limit error, got: %v", err)
	}
}

func TestVehicleHandler_MultipartUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userStore := store.NewUserStore()
	vehicleStore := store.NewVehicleStore()
	tokenStore := auth.NewTokenStore()

	_ = userStore.Create(domain.User{ID: "seller-img", Username: "carlos", Role: domain.RoleSeller, ContactInfo: "carlos@mail.com"})
	token := tokenStore.CreateToken("seller-img", domain.RoleSeller)

	r := gin.New()
	vh := handlers.NewVehicleHandler(vehicleStore, userStore)
	r.POST("/api/seller/vehicles", auth.AuthMiddleware(tokenStore), auth.RequireRole(domain.RoleSeller), vh.CreatePublication)
	r.POST("/api/seller/upload", auth.AuthMiddleware(tokenStore), auth.RequireRole(domain.RoleSeller), vh.UploadImage)

	// Subida vía multipart en /api/seller/vehicles
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("title", "Renault Logan")
	_ = writer.WriteField("description", "Impecable estado")
	_ = writer.WriteField("price", "28000000")
	_ = writer.WriteField("year", "2018")

	part, _ := writer.CreateFormFile("photo", "logan.jpg")
	_, _ = part.Write([]byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x01\x00`\x00`\x00\x00\xff\xdb\x00C\x00"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/seller/vehicles", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	pending := vehicleStore.FindPending()
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending publication, got %d", len(pending))
	}
	if !strings.HasPrefix(pending[0].ImageURL, "/uploads/car-") {
		t.Errorf("expected ImageURL to be set, got: %s", pending[0].ImageURL)
	}

	// Limpieza de la imagen guardada en uploads/
	handlers.DeleteUploadedImage(pending[0].ImageURL, "uploads")
}
