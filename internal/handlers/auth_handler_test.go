package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	valid := `{"username":"juan","password":"SuperSeguro#2026","contact_info":"juan@mail.com","role":"admin"}`

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
	if u.ID == "" || u.PasswordHash == "SuperSeguro#2026" || !auth.CheckPassword("SuperSeguro#2026", u.Salt, u.PasswordHash) {
		t.Error("user must have an ID and a salted password hash")
	}

	if w := post(valid); w.Code != http.StatusBadRequest {
		t.Errorf("duplicate username: expected 400, got %d", w.Code)
	}
	if w := post(`{"username":"ana","password":"SuperSeguro#2026"}`); w.Code != http.StatusBadRequest {
		t.Errorf("missing contact_info: expected 400, got %d", w.Code)
	}
	if w := post(`not json`); w.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON: expected 400, got %d", w.Code)
	}

	// Password policy validations
	short := post(`{"username":"ana","password":"short","contact_info":"ana@mail.com"}`)
	if short.Code != http.StatusBadRequest || !strings.Contains(short.Body.String(), auth.ErrPasswordTooShort.Error()) {
		t.Errorf("expected 400 with short password error, got %d: %s", short.Code, short.Body.String())
	}

	containsUser := post(`{"username":"carlos","password":"carlos123456","contact_info":"carlos@mail.com"}`)
	if containsUser.Code != http.StatusBadRequest || !strings.Contains(containsUser.Body.String(), auth.ErrPasswordContainsUsername.Error()) {
		t.Errorf("expected 400 with password contains username error, got %d: %s", containsUser.Code, containsUser.Body.String())
	}

	blacklisted := post(`{"username":"elena","password":"qwertyuiop","contact_info":"elena@mail.com"}`)
	if blacklisted.Code != http.StatusBadRequest || !strings.Contains(blacklisted.Body.String(), auth.ErrPasswordBlacklisted.Error()) {
		t.Errorf("expected 400 with blacklisted password error, got %d: %s", blacklisted.Code, blacklisted.Body.String())
	}

	// Login
	w := login(`{"username":"juan","password":"SuperSeguro#2026"}`)
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
	noUser := login(`{"username":"nadie","password":"SuperSeguro#2026"}`)
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

	// Validaciones de política de contraseñas en SetupAdmin
	wShort := httptest.NewRecorder()
	reqShort := httptest.NewRequest(http.MethodPost, "/api/auth/setup-admin", strings.NewReader(`{"username":"superadmin","password":"short"}`))
	reqShort.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wShort, reqShort)
	if wShort.Code != http.StatusBadRequest || !strings.Contains(wShort.Body.String(), auth.ErrPasswordTooShort.Error()) {
		t.Fatalf("expected 400 for short admin password, got %d: %s", wShort.Code, wShort.Body.String())
	}

	wUser := httptest.NewRecorder()
	reqUser := httptest.NewRequest(http.MethodPost, "/api/auth/setup-admin", strings.NewReader(`{"username":"superadmin","password":"superadmin2026!"}`))
	reqUser.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wUser, reqUser)
	if wUser.Code != http.StatusBadRequest || !strings.Contains(wUser.Body.String(), auth.ErrPasswordContainsUsername.Error()) {
		t.Fatalf("expected 400 for admin password containing username, got %d: %s", wUser.Code, wUser.Body.String())
	}

	wBlack := httptest.NewRecorder()
	reqBlack := httptest.NewRequest(http.MethodPost, "/api/auth/setup-admin", strings.NewReader(`{"username":"superadmin","password":"password123"}`))
	reqBlack.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wBlack, reqBlack)
	if wBlack.Code != http.StatusBadRequest || !strings.Contains(wBlack.Body.String(), auth.ErrPasswordBlacklisted.Error()) {
		t.Fatalf("expected 400 for blacklisted admin password, got %d: %s", wBlack.Code, wBlack.Body.String())
	}

	// Registrar primer admin con contraseña válida
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/auth/setup-admin", strings.NewReader(`{"username":"superadmin","password":"AdminSeguro#2026"}`))
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
	req = httptest.NewRequest(http.MethodPost, "/api/auth/setup-admin", strings.NewReader(`{"username":"otroadmin","password":"AdminSeguro#2026","contact_info":"otro@test.com"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoginAccountLockoutLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userStore := store.NewUserStore()
	tokenStore := auth.NewTokenStore()
	h := NewAuthHandler(userStore, tokenStore)

	r := gin.New()
	r.POST("/api/auth/register", h.Register)
	r.POST("/api/auth/login", h.Login)

	login := func(username, password string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		body := `{"username":"` + username + `","password":"` + password + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	// 1. Registrar usuario legítimo
	wReg := httptest.NewRecorder()
	regBody := `{"username":"bloqueo_user","password":"Password123!","contact_info":"bloqueo@test.com"}`
	reqReg := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(regBody))
	reqReg.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wReg, reqReg)
	if wReg.Code != http.StatusCreated {
		t.Fatalf("registro falló: %d - %s", wReg.Code, wReg.Body.String())
	}

	// 2. Realizar 5 intentos con contraseña errónea (deben retornar 401 Unauthorized)
	for i := 1; i <= 5; i++ {
		w := login("bloqueo_user", "clave_incorrecta")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("intento %d: esperaba 401, obtuvo %d", i, w.Code)
		}
	}

	// 3. Intento 6 (con contraseña incorrecta): la cuenta debe estar bloqueada con 423 Locked
	wLock := login("bloqueo_user", "otra_clave_incorrecta")
	if wLock.Code != http.StatusLocked {
		t.Fatalf("esperaba 423 StatusLocked en intento 6, obtuvo %d: %s", wLock.Code, wLock.Body.String())
	}
	if !strings.Contains(wLock.Body.String(), "bloqueada temporalmente") {
		t.Errorf("respuesta 423 debe incluir mensaje descriptivo, got: %s", wLock.Body.String())
	}

	// 4. Intento 7 con la CONTRASEÑA CORRECTA: debe ser rechazado con 423 Locked mientras persista el bloqueo
	wCorrectWhileLocked := login("bloqueo_user", "Password123!")
	if wCorrectWhileLocked.Code != http.StatusLocked {
		t.Fatalf("esperaba 423 StatusLocked incluso con password correcta mientras esté bloqueado, obtuvo %d", wCorrectWhileLocked.Code)
	}

	// 5. Configurar política de corta duración para validar desbloqueo y reset
	userStore.SetLockoutPolicy(5, 50*time.Millisecond)
	// Forzar 5 fallos bajo la nueva política
	for i := 0; i < 5; i++ {
		userStore.RecordFailedAttempt("bloqueo_user")
	}
	// Esperar que transcurra el tiempo de bloqueo
	time.Sleep(60 * time.Millisecond)

	// 6. Ahora con contraseña correcta debe ingresar exitosamente (200 OK) y resetear el contador
	wSuccess := login("bloqueo_user", "Password123!")
	if wSuccess.Code != http.StatusOK {
		t.Fatalf("tras expiración, login con contraseña correcta debió ser 200 OK, obtuvo %d: %s", wSuccess.Code, wSuccess.Body.String())
	}

	// 7. El contador debe haberse reseteado a 0
	if userStore.GetFailedAttempts("bloqueo_user") != 0 {
		t.Fatalf("el contador de fallos debió resetearse a 0 tras login exitoso, got %d", userStore.GetFailedAttempts("bloqueo_user"))
	}
}
