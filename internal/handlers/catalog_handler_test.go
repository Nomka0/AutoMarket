package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

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
