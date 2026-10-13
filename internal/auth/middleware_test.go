package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := NewTokenStore()
	store.Set("valid", "user-1")

	r := gin.New()
	r.GET("/", AuthMiddleware(store), func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("userID"))
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
			if tc.status == http.StatusOK && w.Body.String() != "user-1" {
				t.Fatalf("userID = %q, se esperaba user-1", w.Body.String())
			}
		})
	}
}
