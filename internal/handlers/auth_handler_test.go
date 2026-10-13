package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"automarket/internal/auth"
	"automarket/internal/domain"
	"automarket/internal/store"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userStore := store.NewUserStore()
	r := gin.New()
	r.POST("/api/auth/register", NewAuthHandler(userStore).Register)

	post := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	// role enviado por el cliente debe ignorarse
	valid := `{"username":"juan","password":"secreto","contact_info":"juan@mail.com","role":"admin"}`

	if w := post(valid); w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	u, ok := userStore.FindByUsername("juan")
	if !ok {
		t.Fatal("user was not stored")
	}
	if u.Role != domain.RoleSeller {
		t.Errorf("expected role %q, got %q", domain.RoleSeller, u.Role)
	}
	if u.ID == "" || u.PasswordHash == "secreto" || !auth.CheckPassword("secreto", u.Salt, u.PasswordHash) {
		t.Error("user must have an ID and a salted password hash")
	}

	if w := post(valid); w.Code != http.StatusBadRequest {
		t.Errorf("duplicate username: expected 400, got %d", w.Code)
	}
	if w := post(`{"username":"ana","password":"secreto"}`); w.Code != http.StatusBadRequest {
		t.Errorf("missing contact_info: expected 400, got %d", w.Code)
	}
	if w := post(`not json`); w.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON: expected 400, got %d", w.Code)
	}
}
