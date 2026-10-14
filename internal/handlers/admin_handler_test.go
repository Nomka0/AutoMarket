package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"automarket/internal/domain"
	"automarket/internal/store"
)

func TestGetPendingPublications(t *testing.T) {
	gin.SetMode(gin.TestMode)
	vehicleStore := store.NewVehicleStore()
	r := gin.New()
	r.GET("/pending", NewAdminHandler(vehicleStore).GetPendingPublications)

	get := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/pending", nil))
		return w
	}

	if w := get(); w.Code != http.StatusOK || w.Body.String() != "[]" {
		t.Errorf("empty pending list: expected 200 [], got %d %s", w.Code, w.Body.String())
	}

	vehicleStore.Create(domain.VehiclePublication{ID: "p1", Status: domain.StatusPending})
	vehicleStore.Create(domain.VehiclePublication{ID: "p2", Status: domain.StatusPending})
	vehicleStore.Create(domain.VehiclePublication{ID: "approved", Status: domain.StatusApproved})
	vehicleStore.Create(domain.VehiclePublication{ID: "sold", Status: domain.StatusSold})

	w := get()
	var pubs []domain.VehiclePublication
	if err := json.Unmarshal(w.Body.Bytes(), &pubs); err != nil || w.Code != http.StatusOK {
		t.Fatalf("expected 200 with a JSON list, got %d %s", w.Code, w.Body.String())
	}
	if len(pubs) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pubs))
	}
	for _, p := range pubs {
		if p.Status != domain.StatusPending {
			t.Errorf("publication %s has status %s, only PENDING allowed", p.ID, p.Status)
		}
	}
}

func TestApprovePublication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	vehicleStore := store.NewVehicleStore()
	r := gin.New()
	r.PATCH("/:id/approve", NewAdminHandler(vehicleStore).ApprovePublication)

	approve := func(id string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/"+id+"/approve", nil))
		return w
	}

	vehicleStore.Create(domain.VehiclePublication{ID: "p1", Title: "Mazda 3", Status: domain.StatusPending})
	vehicleStore.Create(domain.VehiclePublication{ID: "sold", Status: domain.StatusSold})

	w := approve("p1")
	var pub domain.VehiclePublication
	if err := json.Unmarshal(w.Body.Bytes(), &pub); err != nil || w.Code != http.StatusOK {
		t.Fatalf("approve: expected 200 with the publication, got %d %s", w.Code, w.Body.String())
	}
	if pub.ID != "p1" || pub.Title != "Mazda 3" || pub.Status != domain.StatusApproved {
		t.Errorf("unexpected response: %+v", pub)
	}
	// HU-ADM-01: aprobada sale de pendientes y queda en lo que lee el catalogo
	if approved := vehicleStore.FindApproved(); len(approved) != 1 || approved[0].ID != "p1" || len(vehicleStore.FindPending()) != 0 {
		t.Error("approved publication must leave pending and appear in approved")
	}

	if w := approve("p1"); w.Code != http.StatusConflict {
		t.Errorf("approve twice: expected 409, got %d", w.Code)
	}
	if w := approve("sold"); w.Code != http.StatusConflict {
		t.Errorf("approve sold: expected 409, got %d", w.Code)
	}
	if s, _ := vehicleStore.FindByID("sold"); s.Status != domain.StatusSold {
		t.Errorf("sold publication must stay SOLD, got %s", s.Status)
	}
	if w := approve("nope"); w.Code != http.StatusNotFound {
		t.Errorf("approve missing: expected 404, got %d", w.Code)
	}
}

func TestDeletePublication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	vehicleStore := store.NewVehicleStore()
	r := gin.New()
	r.DELETE("/:id", NewAdminHandler(vehicleStore).DeletePublication)

	del := func(id string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/"+id, nil))
		return w
	}

	vehicleStore.Create(domain.VehiclePublication{ID: "p1", Status: domain.StatusPending})
	vehicleStore.Create(domain.VehiclePublication{ID: "p2", Status: domain.StatusApproved})

	if w := del("p1"); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"message"`) {
		t.Errorf("delete: expected 200 with message, got %d %s", w.Code, w.Body.String())
	}
	if _, ok := vehicleStore.FindByID("p1"); ok {
		t.Error("deleted publication still stored")
	}
	if _, ok := vehicleStore.FindByID("p2"); !ok {
		t.Error("other publications must not be deleted")
	}
	if w := del("p1"); w.Code != http.StatusNotFound {
		t.Errorf("delete twice: expected 404, got %d", w.Code)
	}
}

func TestDeletePublicationConcurrent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	vehicleStore := store.NewVehicleStore()
	r := gin.New()
	r.DELETE("/:id", NewAdminHandler(vehicleStore).DeletePublication)
	vehicleStore.Create(domain.VehiclePublication{ID: "p1", Status: domain.StatusPending})

	// 20 borrados simultaneos de la misma publicacion: exactamente uno gana
	codes := make(chan int, 20)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/p1", nil))
			codes <- w.Code
		})
	}
	wg.Wait()
	close(codes)

	ok := 0
	for code := range codes {
		if code == http.StatusOK {
			ok++
		} else if code != http.StatusNotFound {
			t.Errorf("unexpected status %d", code)
		}
	}
	if ok != 1 {
		t.Errorf("expected exactly one 200, got %d", ok)
	}
}
