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
