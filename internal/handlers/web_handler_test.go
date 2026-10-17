package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"automarket/internal/auth"
	"automarket/internal/domain"
	"automarket/internal/store"
)

func setupTestRouter() (*gin.Engine, *store.UserStore, *store.VehicleStore, *auth.TokenStore) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.LoadHTMLGlob("../../templates/*")

	userStore := store.NewUserStore()
	vehicleStore := store.NewVehicleStore()
	tokenStore := auth.NewTokenStore()
	webHandler := NewWebHandler(userStore, vehicleStore, tokenStore)

	r.GET("/", webHandler.Index)
	r.GET("/catalog", auth.WebOptionalAuthMiddleware(tokenStore), webHandler.Catalog)
	r.GET("/catalog/:id", auth.WebOptionalAuthMiddleware(tokenStore), webHandler.CatalogDetail)
	r.GET("/login", auth.WebOptionalAuthMiddleware(tokenStore), webHandler.LoginPage)
	r.POST("/login", webHandler.LoginSubmit)
	r.GET("/logout", webHandler.Logout)
	r.GET("/register", webHandler.RegisterPage)
	r.POST("/register", webHandler.RegisterSubmit)
	r.GET("/setup-admin", webHandler.SetupAdminPage)
	r.POST("/setup-admin", webHandler.SetupAdminSubmit)

	sellerWeb := r.Group("/seller").Use(auth.WebAuthMiddleware(tokenStore), auth.WebRequireRole(domain.RoleSeller))
	sellerWeb.GET("", webHandler.SellerDashboard)
	sellerWeb.POST("/vehicles", webHandler.CreateVehicleSubmit)
	sellerWeb.POST("/vehicles/:id/sold", webHandler.MarkSoldSubmit)
	sellerWeb.POST("/contact", webHandler.UpdateContactSubmit)

	adminWeb := r.Group("/admin").Use(auth.WebAuthMiddleware(tokenStore), auth.WebRequireRole(domain.RoleAdmin))
	adminWeb.GET("", webHandler.AdminDashboard)
	adminWeb.POST("/publications/:id/approve", webHandler.ApprovePublicationSubmit)
	adminWeb.POST("/publications/:id/delete", webHandler.DeletePublicationSubmit)
	adminWeb.POST("/users/:id/delete", webHandler.DeleteUserSubmit)

	return r, userStore, vehicleStore, tokenStore
}

func TestWebHandler_CatalogAndVisitor(t *testing.T) {
	r, _, vehicleStore, _ := setupTestRouter()

	vehicleStore.Create(domain.VehiclePublication{
		ID:            "veh-1",
		Title:         "Mazda 3",
		Price:         45000000,
		Year:          2020,
		Status:        domain.StatusApproved,
		SellerContact: "vendedor@test.com",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Mazda 3") {
		t.Error("catalog HTML should contain vehicle title")
	}
	if !strings.Contains(w.Body.String(), "vendedor@test.com") {
		t.Error("catalog HTML should display seller contact")
	}

	// Detalle
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/catalog/veh-1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Mazda 3") {
		t.Fatalf("expected 200 on detail, got %d", w.Code)
	}
}

func TestWebHandler_SellerFlow(t *testing.T) {
	r, userStore, vehicleStore, _ := setupTestRouter()

	// 1. Registro de vendedor
	formData := url.Values{
		"username":     {"pedro"},
		"password":     {"pass123"},
		"contact_info": {"3101112233"},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", w.Code)
	}
	cookie := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "token=") {
		t.Fatal("expected auth cookie after registration")
	}

	// Extraer cookie token
	var tokenCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "token" {
			tokenCookie = c
			break
		}
	}
	if tokenCookie == nil {
		t.Fatal("token cookie not found")
	}

	// 2. Acceso a dashboard de vendedor con cookie
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/seller", nil)
	req.AddCookie(tokenCookie)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on seller dashboard, got %d", w.Code)
	}

	// 3. Publicar vehículo desde web
	vehForm := url.Values{
		"title":       {"Chevrolet Onix"},
		"description": {"Excelente estado"},
		"price":       {"35000000"},
		"year":        {"2019"},
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/seller/vehicles", strings.NewReader(vehForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(tokenCookie)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect after publishing, got %d", w.Code)
	}

	// Verificar en store que el vehículo quedó en PENDING
	pending := vehicleStore.FindPending()
	if len(pending) != 1 || pending[0].Title != "Chevrolet Onix" {
		t.Fatalf("expected pending vehicle in store, got %d", len(pending))
	}

	// 4. Intentar entrar como Admin siendo Vendedor -> 403 Forbidden
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(tokenCookie)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 forbidden accessing admin with seller role, got %d", w.Code)
	}

	// 5. Logout
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(tokenCookie)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect after logout, got %d", w.Code)
	}
	_ = userStore
}

func TestWebHandler_AdminFlow(t *testing.T) {
	r, _, vehicleStore, _ := setupTestRouter()

	// 1. Setup Admin inicial
	adminForm := url.Values{
		"username": {"adminweb"},
		"password": {"adminpass123"},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/setup-admin", strings.NewReader(adminForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect after setup-admin, got %d", w.Code)
	}

	var adminCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "token" {
			adminCookie = c
			break
		}
	}
	if adminCookie == nil {
		t.Fatal("admin token cookie not found")
	}

	// Crear vehículo pendiente
	vehicleStore.Create(domain.VehiclePublication{
		ID:     "veh-pending",
		Title:  "Renault Kwid",
		Status: domain.StatusPending,
	})

	// 2. Dashboard admin
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(adminCookie)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on admin dashboard, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Renault Kwid") {
		t.Error("admin dashboard should list pending publication")
	}

	// 3. Aprobar vehículo
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/admin/publications/veh-pending/approve", nil)
	req.AddCookie(adminCookie)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 after approve, got %d", w.Code)
	}

	// Verificar aprobado
	pub, _ := vehicleStore.FindByID("veh-pending")
	if pub.Status != domain.StatusApproved {
		t.Fatalf("expected status APPROVED, got %s", pub.Status)
	}
}

func TestWebHandler_CatalogDetailContactAndOneClick(t *testing.T) {
	r, _, vehicleStore, _ := setupTestRouter()

	vehicleStore.Create(domain.VehiclePublication{
		ID:            "veh-contact-1",
		Title:         "Mazda 3 Grand Touring",
		Price:         55000000,
		Year:          2022,
		Status:        domain.StatusApproved,
		SellerContact: "juan@gmail.com / 3001234567",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/catalog/veh-contact-1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	requiredTexts := []string{
		"Información de Contacto del Vendedor:",
		"Correo:",
		"juan@gmail.com",
		"href=\"mailto:juan@gmail.com?subject=",
		"Número de telefono:",
		"&#43;573001234567",
		"href=\"tel:&#43;573001234567\"",
		"https://wa.me/573001234567?text=",
	}

	for _, text := range requiredTexts {
		if !strings.Contains(body, text) {
			t.Errorf("catalog detail body missing required text %q", text)
		}
	}
}

func TestWebHandler_SellerUpdateContact(t *testing.T) {
	r, userStore, vehicleStore, tokenStore := setupTestRouter()

	salt := auth.GenerateSalt()
	_ = userStore.Create(domain.User{
		ID:           "seller-update-1",
		Username:     "seller1",
		PasswordHash: auth.HashPassword("secret", salt),
		Salt:         salt,
		Role:         domain.RoleSeller,
		ContactInfo:  "juan@gmail.com",
	})

	vehicleStore.Create(domain.VehiclePublication{
		ID:            "veh-seller-1",
		SellerID:      "seller-update-1",
		Title:         "Toyota Corolla",
		Price:         40000000,
		Year:          2021,
		Status:        domain.StatusApproved,
		SellerContact: "juan@gmail.com",
	})

	token := tokenStore.CreateToken("seller-update-1", domain.RoleSeller)

	form := url.Values{}
	form.Set("email", "juan@gmail.com")
	form.Set("phone", "3209876543")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/seller/contact", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", w.Code)
	}

	u, _ := userStore.FindByID("seller-update-1")
	if u.GetPhone() != "3209876543" {
		t.Errorf("expected phone 3209876543, got %q", u.GetPhone())
	}
	if u.GetEmail() != "juan@gmail.com" {
		t.Errorf("expected email juan@gmail.com, got %q", u.GetEmail())
	}

	v, _ := vehicleStore.FindByID("veh-seller-1")
	if v.ContactPhone() != "3209876543" {
		t.Errorf("expected vehicle to have updated phone, got %q", v.ContactPhone())
	}
}

