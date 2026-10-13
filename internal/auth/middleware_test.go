package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := NewTokenStore()
	store.Set("valid", "user-1", "vendedor")
	store.tokens["expired"] = Session{UserID: "user-1", ExpiresAt: time.Now().Add(-time.Minute)}

	r := gin.New()
	r.GET("/", AuthMiddleware(store), func(c *gin.Context) {
		session, ok := GetCurrentUser(c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.String(http.StatusOK, session.UserID+":"+session.Role)
	})

	cases := []struct {
		name, header string
		status       int
	}{
		{"sin header", "", http.StatusUnauthorized},
		{"sin prefijo Bearer", "valid", http.StatusUnauthorized},
		{"esquema distinto", "Basic valid", http.StatusUnauthorized},
		{"Bearer sin token", "Bearer ", http.StatusUnauthorized},
		{"token desconocido", "Bearer nope", http.StatusUnauthorized},
		{"token expirado", "Bearer expired", http.StatusUnauthorized},
		{"token válido", "Bearer valid", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status = %d, se esperaba %d", w.Code, tc.status)
			}
			if tc.status == http.StatusOK && w.Body.String() != "user-1:vendedor" {
				t.Fatalf("sesión = %q, se esperaba user-1:vendedor", w.Body.String())
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := NewTokenStore()
	store.Set("seller", "user-1", "vendedor")
	store.Set("admin", "user-2", "admin")

	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r := gin.New()
	r.GET("/admin", AuthMiddleware(store), RequireRole("admin"), ok)
	r.GET("/both", AuthMiddleware(store), RequireRole("admin", "vendedor"), ok)
	r.GET("/sin-auth", RequireRole("admin"), ok)

	cases := []struct {
		name, path, token string
		status            int
	}{
		{"rol permitido", "/admin", "admin", http.StatusOK},
		{"rol no permitido", "/admin", "seller", http.StatusForbidden},
		{"varios roles permitidos", "/both", "seller", http.StatusOK},
		{"sin token", "/admin", "", http.StatusUnauthorized},
		{"sin AuthMiddleware previo", "/sin-auth", "admin", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status = %d, se esperaba %d", w.Code, tc.status)
			}
		})
	}
}
