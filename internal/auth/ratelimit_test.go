package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewIPRateLimiter(5, 1*time.Minute)
	r := gin.New()
	r.POST("/api/auth/login", limiter.Middleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	performRequest := func(ip string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.Header.Set("X-Forwarded-For", ip)
		req.RemoteAddr = ip + ":12345"
		r.ServeHTTP(w, req)
		return w
	}

	clientIP := "192.168.1.100"

	// Las primeras 5 peticiones deben ser permitidas (HTTP 200)
	for i := 1; i <= 5; i++ {
		w := performRequest(clientIP)
		if w.Code != http.StatusOK {
			t.Fatalf("petición %d: esperaba código 200, obtuvo %d", i, w.Code)
		}
	}

	// La 6ta petición debe ser bloqueada (HTTP 429)
	w6 := performRequest(clientIP)
	if w6.Code != http.StatusTooManyRequests {
		t.Fatalf("petición 6: esperaba código 429, obtuvo %d", w6.Code)
	}

	retryAfter := w6.Header().Get("Retry-After")
	if retryAfter != "60" {
		t.Errorf("esperaba cabecera Retry-After: 60, obtuvo: %q", retryAfter)
	}

	// Otra IP distinta debe tener su propio contador independiente
	otherIP := "192.168.1.200"
	wOther := performRequest(otherIP)
	if wOther.Code != http.StatusOK {
		t.Fatalf("otra IP: esperaba código 200, obtuvo %d", wOther.Code)
	}
}
