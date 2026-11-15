package handlers

import (
	"encoding/base64"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"

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

func TestAdminLoginRequiresValidTOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userStore := store.NewUserStore()
	const secret = "JBSWY3DPEHPK3PXP"
	admin, err := userStore.CreateAdminWithTOTP("admin", "password123", "admin@example.com", secret)
	if err != nil {
		t.Fatal(err)
	}

	tokenStore := auth.NewTokenStore()
	r := gin.New()
	r.POST("/api/auth/login", NewAuthHandler(userStore, tokenStore).Login)
	login := func(totpCode string, includeCode bool) *httptest.ResponseRecorder {
		body := `{"username":"admin","password":"password123"`
		if includeCode {
			body += `,"totp_code":"` + totpCode + `"`
		}
		body += `}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	validCode, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	validResponse := login(validCode, true)
	if validResponse.Code != http.StatusOK {
		t.Fatalf("valid TOTP: expected 200, got %d: %s", validResponse.Code, validResponse.Body.String())
	}
	var response struct {
		Token string `json:"token"`
		Role  string `json:"role"`
	}
	if err := json.Unmarshal(validResponse.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Token == "" || response.Role != domain.RoleAdmin {
		t.Fatalf("expected authenticated admin response, got %+v", response)
	}
	if session, ok := tokenStore.ValidateToken(response.Token); !ok || session.UserID != admin.ID {
		t.Fatal("valid TOTP login did not create an admin session")
	}

	invalidCode := []byte(validCode)
	invalidCode[len(invalidCode)-1] = '0' + (invalidCode[len(invalidCode)-1]-'0'+1)%10
	for _, testCase := range []struct {
		name        string
		code        string
		includeCode bool
	}{
		{name: "invalid code", code: string(invalidCode), includeCode: true},
		{name: "missing code"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			w := login(testCase.code, testCase.includeCode)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
			}
			var errorResponse map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err != nil {
				t.Fatal(err)
			}
			if errorResponse["error"] != "código 2FA TOTP inválido o requerido para administradores" {
				t.Errorf("unexpected error response: %s", w.Body.String())
			}
		})
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
	var enrollment struct {
		Secret  string `json:"secret"`
		OTPAuth string `json:"otpauth_url"`
		QRCode  string `json:"qr_code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &enrollment); err != nil {
		t.Fatal(err)
	}
	if enrollment.Secret == "" || !strings.HasPrefix(enrollment.OTPAuth, "otpauth://totp/") {
		t.Fatalf("expected TOTP secret and provisioning URL, got %+v", enrollment)
	}
	pngBytes, err := base64.StdEncoding.DecodeString(enrollment.QRCode)
	if err != nil {
		t.Fatalf("QR code is not valid base64: %v", err)
	}
	if _, err := png.Decode(strings.NewReader(string(pngBytes))); err != nil {
		t.Fatalf("QR code is not a PNG image: %v", err)
	}
	admin, ok := userStore.FindByUsername("superadmin")
	if !ok || admin.TOTPSecret != enrollment.Secret {
		t.Fatal("TOTP secret was not saved on the administrator")
	}
	code, err := totp.GenerateCode(admin.TOTPSecret, time.Now())
	if err != nil || !totp.Validate(code, admin.TOTPSecret) {
		t.Fatal("enrollment secret must produce a valid RFC 6238 code")
	}
	if strings.Contains(w.Body.String(), "data:image/png") {
		t.Error("QR code should be returned as raw base64, not a data URI")
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
