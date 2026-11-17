package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"

	"golang.org/x/crypto/argon2"
)

// Parámetros recomendados por OWASP para Argon2id:
// Memoria: 64 MB (65536 KiB), Iteraciones: 3, Hilos/Paralelismo: 4, Longitud de Clave: 32 bytes
const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
)

// GenerateSalt genera un salt criptográficamente seguro de 16 bytes en formato hexadecimal (32 caracteres).
func GenerateSalt() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// HashPassword deriva un hash seguro usando Argon2id con parámetros recomendados por OWASP.
// Si salt está vacío, genera automáticamente uno nuevo.
func HashPassword(password, salt string) string {
	var saltBytes []byte
	if salt == "" {
		salt = GenerateSalt()
	}

	decoded, err := hex.DecodeString(salt)
	if err == nil && len(decoded) >= 16 {
		saltBytes = decoded
	} else {
		saltBytes = []byte(salt)
	}

	hash := argon2.IDKey([]byte(password), saltBytes, argonTime, argonMemory, argonThreads, argonKeyLen)
	return hex.EncodeToString(hash)
}

// CheckPassword verifica la contraseña derivando el hash con Argon2id y comparándolo en tiempo constante.
func CheckPassword(password, salt, hash string) bool {
	computed := HashPassword(password, salt)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}

// VerifyPassword es un alias de CheckPassword para mantener compatibilidad con el resto del backend.
func VerifyPassword(password, salt, hash string) bool {
	return CheckPassword(password, salt, hash)
}

// GenerateToken genera un token opaco criptográfico aleatorio de 32 bytes en hexadecimal (64 caracteres).
func GenerateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
