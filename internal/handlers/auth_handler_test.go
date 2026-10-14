package handlers

import (
	"encoding/json"
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
	tokenStore := auth.NewTokenStore()
	h := NewAuthHandler(userStore, tokenStore)
	r.POST("/api/auth/register", h.Register)
	r.POST("/api/auth/login", h.Login)

	postTo := func(path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}
	post := func(body string) *httptest.ResponseRecorder { return postTo("/api/auth/register", body) }
	login := func(body string) *httptest.ResponseRecorder { return postTo("/api/auth/login", body) }

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

	// Login
	w := login(`{"username":"juan","password":"secreto"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct{ Token, Role, Username string }
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Role != domain.RoleSeller || resp.Username != "juan" {
		t.Errorf("unexpected login response: %+v", resp)
	}
	if s, ok := tokenStore.ValidateToken(resp.Token); !ok || s.UserID != u.ID || s.Role != u.Role {
		t.Error("login token must map to a valid session for the user")
	}

	wrongPass := login(`{"username":"juan","password":"mala"}`)
	noUser := login(`{"username":"nadie","password":"secreto"}`)
	if wrongPass.Code != http.StatusUnauthorized || noUser.Code != http.StatusUnauthorized {
		t.Errorf("bad credentials: expected 401, got %d and %d", wrongPass.Code, noUser.Code)
	}
	if wrongPass.Body.String() != noUser.Body.String() {
		t.Error("401 responses must not reveal whether the user exists")
	}
	if w := login(`{"username":"juan"}`); w.Code != http.StatusBadRequest {
		t.Errorf("missing password: expected 400, got %d", w.Code)
	}
}

func TestSetupAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userStore := store.NewUserStore()
	r := gin.New()
	tokenStore := auth.NewTokenStore()
	h := NewAuthHandler(userStore, tokenStore)
	r.POST("/api/auth/setup-admin", h.SetupAdmin)
	r.GET("/api/auth/has-admin", h.HasAdmin)

	// Inicialmente no hay admin
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/has-admin", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"has_admin":false`) {
		t.Fatalf("expected has_admin false, got: %s", w.Body.String())
	}

	// Registrar primer admin
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/auth/setup-admin", strings.NewReader(`{"username":"superadmin","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// has-admin ahora debe ser true
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/auth/has-admin", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"has_admin":true`) {
		t.Fatalf("expected has_admin true, got: %s", w.Body.String())
	}

	// Intentar registrar segundo admin -> 403 Forbidden
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/auth/setup-admin", strings.NewReader(`{"username":"otroadmin","password":"password123","contact_info":"otro@test.com"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
