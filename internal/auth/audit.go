package auth

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditLogEntry representa un registro estructurado de auditoría para trazabilidad según RF-01 Accounting.
type AuditLogEntry struct {
	Timestamp  string `json:"timestamp"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	IP         string `json:"ip"`
	UserID     string `json:"user_id"`
	Role       string `json:"role"`
	StatusCode int    `json:"status_code"`
	DurationMs int64  `json:"duration_ms"`
}

// AuditLogger administra la escritura thread-safe de registros de auditoría en formato JSON Lines.
type AuditLogger struct {
	mu   sync.Mutex
	file *os.File
}

func NewAuditLogger(filename ...string) (*AuditLogger, error) {
	name := "audit.log"
	if len(filename) > 0 && filename[0] != "" {
		name = filename[0]
	}
	file, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &AuditLogger{file: file}, nil
}

func NewFileAuditLogger(filePath string) (*AuditLogger, error) {
	return NewAuditLogger(filePath)
}

func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *AuditLogger) Log(entry AuditLogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err = l.file.Write(append(data, '\n'))
	return err
}

// AuditLogMiddleware intercepta cada petición y respuesta para registrarla en el AuditLogger.
func AuditLogMiddleware(logger *AuditLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next() // Procesar la petición primero
		duration := time.Since(start).Milliseconds()

		if logger == nil {
			return
		}

		userID := "anonymous"
		role := "visitor"
		if session, ok := GetCurrentUser(c); ok {
			if session.UserID != "" {
				userID = session.UserID
			}
			if session.Role != "" {
				role = session.Role
			}
		}

		entry := AuditLogEntry{
			Timestamp:  start.UTC().Format(time.RFC3339),
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			IP:         c.ClientIP(),
			UserID:     userID,
			Role:       role,
			StatusCode: c.Writer.Status(),
			DurationMs: duration,
		}

		_ = logger.Log(entry)
	}
}
