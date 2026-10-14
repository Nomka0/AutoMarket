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

func TestGetCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	vehicleStore := store.NewVehicleStore()
	r := gin.New()
	r.GET("/api/catalog", NewCatalogHandler(vehicleStore).GetCatalog)

	get := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/catalog", nil))
		return w
	}

	if w := get(); w.Code != http.StatusOK || w.Body.String() != "[]" {
		t.Errorf("empty catalog: expected 200 [], got %d %s", w.Code, w.Body.String())
	}

	vehicleStore.Create(domain.VehiclePublication{ID: "a1", Status: domain.StatusApproved, SellerContact: "juan@mail.com"})
	vehicleStore.Create(domain.VehiclePublication{ID: "a2", Status: domain.StatusApproved})
	vehicleStore.Create(domain.VehiclePublication{ID: "pending", Status: domain.StatusPending})
	vehicleStore.Create(domain.VehiclePublication{ID: "sold", Status: domain.StatusSold})

	w := get()
	var pubs []domain.VehiclePublication
	if err := json.Unmarshal(w.Body.Bytes(), &pubs); err != nil || w.Code != http.StatusOK {
		t.Fatalf("expected 200 with a JSON list, got %d %s", w.Code, w.Body.String())
	}
	if len(pubs) != 2 {
		t.Fatalf("expected 2 approved, got %d", len(pubs))
	}
	for _, p := range pubs {
		if p.Status != domain.StatusApproved {
			t.Errorf("publication %s has status %s, only APPROVED allowed", p.ID, p.Status)
		}
		if p.ID == "a1" && p.SellerContact != "juan@mail.com" {
			t.Errorf("catalog must include seller contact, got %q", p.SellerContact)
		}
	}
}

// HU-VIS-01: flujo real vendedor -> admin -> visitante; el catalogo expone
// los datos del vehiculo y el contacto que el vendedor dio al registrarse
func TestCatalogIncludesVehicleDataAndSellerContact(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userStore := store.NewUserStore()
	vehicleStore := store.NewVehicleStore()
	tokenStore := auth.NewTokenStore()
	_ = userStore.Create(domain.User{ID: "seller-1", Username: "juan", Role: domain.RoleSeller, ContactInfo: "juan@mail.com / 3001234567"})

	r := gin.New()
	r.POST("/publications", auth.AuthMiddleware(tokenStore), NewVehicleHandler(vehicleStore, userStore).CreatePublication)
	r.PATCH("/publications/:id/approve", NewAdminHandler(vehicleStore).ApprovePublication)
	r.GET("/api/catalog", NewCatalogHandler(vehicleStore).GetCatalog)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"title":"Mazda 3","description":"Unico dueno","price":45000000,"year":2019}`))
	req.Header.Set("Authorization", "Bearer "+tokenStore.CreateToken("seller-1", domain.RoleSeller))
	r.ServeHTTP(w, req)
	var created domain.VehiclePublication
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/publications/"+created.ID+"/approve", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/catalog", nil))
	// se decodifica a mapas para comprobar las claves JSON reales, no los campos del struct
	var items []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil || len(items) != 1 {
		t.Fatalf("expected 1 catalog item, got %d %s", w.Code, w.Body.String())
	}
	want := map[string]any{
		"title":          "Mazda 3",
		"description":    "Unico dueno",
		"price":          45000000.0,
		"year":           2019.0,
		"seller_contact": "juan@mail.com / 3001234567",
	}
	for key, value := range want {
		if items[0][key] != value {
			t.Errorf("catalog item %q: expected %v, got %v", key, value, items[0][key])
		}
	}
}
