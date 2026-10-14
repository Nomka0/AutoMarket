package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuditLogEntryJSON(t *testing.T) {
	entry := AuditLogEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Method:     "GET",
		Path:       "/api/catalog",
		IP:         "127.0.0.1",
		UserID:     "anonymous",
		Role:       "visitor",
		StatusCode: 200,
		DurationMs: 12,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal AuditLogEntry: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	expectedKeys := []string{
		"timestamp",
		"method",
		"path",
		"ip",
		"user_id",
		"role",
		"status_code",
		"duration_ms",
	}

	for _, k := range expectedKeys {
		if _, ok := decoded[k]; !ok {
			t.Errorf("missing expected JSON key: %s", k)
		}
	}
}

func TestAuditLogMiddleware_PublicAndAuthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")
	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("failed to create AuditLogger: %v", err)
	}
	defer logger.Close()

	tokenStore := NewTokenStore()
	token := tokenStore.CreateToken("seller-42", "vendedor")

	r := gin.New()
	r.Use(AuditLogMiddleware(logger))

	r.GET("/public", func(c *gin.Context) {
		c.String(http.StatusOK, "ok public")
	})

	r.GET("/protected", AuthMiddleware(tokenStore), func(c *gin.Context) {
		c.String(http.StatusOK, "ok protected")
	})

	// 1. Petición pública
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/public", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}

	// 2. Petición autenticada
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(lines))
	}

	var entry1, entry2 AuditLogEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry1); err != nil {
		t.Fatalf("failed to parse log line 1: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &entry2); err != nil {
		t.Fatalf("failed to parse log line 2: %v", err)
	}

	if entry1.UserID != "anonymous" || entry1.Role != "visitor" || entry1.Path != "/public" || entry1.StatusCode != 200 {
		t.Errorf("unexpected public entry: %+v", entry1)
	}
	if entry2.UserID != "seller-42" || entry2.Role != "vendedor" || entry2.Path != "/protected" || entry2.StatusCode != 200 {
		t.Errorf("unexpected protected entry: %+v", entry2)
	}
	if entry1.DurationMs < 0 || entry2.DurationMs < 0 {
		t.Errorf("duration_ms must be non-negative: %d, %d", entry1.DurationMs, entry2.DurationMs)
	}
}

func TestAuditLogger_Concurrency(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit_concurrent.log")
	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("failed to create AuditLogger: %v", err)
	}
	defer logger.Close()

	var wg sync.WaitGroup
	count := 50
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(id int) {
			defer wg.Done()
			_ = logger.Log(AuditLogEntry{
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				Method:     "POST",
				Path:       "/api/test",
				IP:         "127.0.0.1",
				UserID:     "user-1",
				Role:       "admin",
				StatusCode: 200,
				DurationMs: int64(id),
			})
		}(i)
	}

	wg.Wait()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != count {
		t.Fatalf("expected %d lines, got %d", count, len(lines))
	}
}
