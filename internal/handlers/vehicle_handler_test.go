package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"automarket/internal/auth"
	"automarket/internal/domain"
	"automarket/internal/store"
)

func TestCreatePublicationRequestValidation(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		valid bool
	}{
		{"valid", `{"title":"Mazda 3","description":"Unico dueno","price":45000000,"year":2019}`, true},
		{"missing title", `{"description":"x","price":1,"year":2019}`, false},
		{"missing description", `{"title":"x","price":1,"year":2019}`, false},
		{"missing price", `{"title":"x","description":"x","year":2019}`, false},
		{"zero price", `{"title":"x","description":"x","price":0,"year":2019}`, false},
		{"negative price", `{"title":"x","description":"x","price":-5,"year":2019}`, false},
		{"year 1900", `{"title":"x","description":"x","price":1,"year":1900}`, false},
		{"price as string", `{"title":"x","description":"x","price":"caro","year":2019}`, false},
		{"decimal year", `{"title":"x","description":"x","price":1,"year":2019.5}`, false},
	}

	for _, tc := range cases {
		var req CreatePublicationRequest
		err := binding.JSON.BindBody([]byte(strings.TrimSpace(tc.body)), &req)
		if (err == nil) != tc.valid {
			t.Errorf("%s: valid=%v, got err=%v", tc.name, tc.valid, err)
		}
	}
}

func TestCreatePublication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userStore := store.NewUserStore()
	vehicleStore := store.NewVehicleStore()
	tokenStore := auth.NewTokenStore()
	_ = userStore.Create(domain.User{ID: "seller-1", Username: "juan", Role: domain.RoleSeller, ContactInfo: "juan@mail.com"})

	r := gin.New()
	r.POST("/api/vehicles", auth.AuthMiddleware(tokenStore), NewVehicleHandler(vehicleStore, userStore).CreatePublication)

	post := func(token, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/vehicles", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)
		return w
	}

	// seller_id y status enviados por el cliente deben ignorarse
	body := `{"title":"Mazda 3","description":"Unico dueno","price":45000000,"year":2019,"status":"APPROVED","seller_id":"otro"}`

	w := post(tokenStore.CreateToken("seller-1", domain.RoleSeller), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.VehiclePublication
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	stored, ok := vehicleStore.FindByID(resp.ID)
	if !ok || resp.ID == "" {
		t.Fatal("publication was not stored")
	}
	if stored.Status != domain.StatusPending || stored.SellerID != "seller-1" || stored.SellerContact != "juan@mail.com" {
		t.Errorf("publication must be PENDING and tied to the seller: %+v", stored)
	}
	if stored.Title != "Mazda 3" || stored.Price != 45000000 || stored.Year != 2019 || stored.CreatedAt.IsZero() {
		t.Errorf("unexpected stored publication: %+v", stored)
	}

	if w := post(tokenStore.CreateToken("seller-1", domain.RoleSeller), `{"title":"x"}`); w.Code != http.StatusBadRequest {
		t.Errorf("invalid body: expected 400, got %d", w.Code)
	}
	for _, blank := range []string{
		`{"title":"   ","description":"Unico dueno","price":45000000,"year":2019}`,
		`{"title":"Mazda 3","description":" \t ","price":45000000,"year":2019}`,
	} {
		if w := post(tokenStore.CreateToken("seller-1", domain.RoleSeller), blank); w.Code != http.StatusBadRequest {
			t.Errorf("blank text %s: expected 400, got %d", blank, w.Code)
		}
	}
	if w := post(tokenStore.CreateToken("borrado", domain.RoleSeller), body); w.Code != http.StatusUnauthorized {
		t.Errorf("deleted user: expected 401, got %d", w.Code)
	}
	if len(vehicleStore.FindPending()) != 1 {
		t.Errorf("only the valid request must be stored, got %d", len(vehicleStore.FindPending()))
	}
}

func TestMarkAsSold(t *testing.T) {
	gin.SetMode(gin.TestMode)
	vehicleStore := store.NewVehicleStore()
	tokenStore := auth.NewTokenStore()
	r := gin.New()
	r.PATCH("/:id/sold", auth.AuthMiddleware(tokenStore), NewVehicleHandler(vehicleStore, store.NewUserStore()).MarkAsSold)

	owner := tokenStore.CreateToken("seller-1", domain.RoleSeller)
	other := tokenStore.CreateToken("seller-2", domain.RoleSeller)
	sold := func(token, id string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/"+id+"/sold", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)
		return w
	}

	vehicleStore.Create(domain.VehiclePublication{ID: "a1", SellerID: "seller-1", Status: domain.StatusApproved})
	vehicleStore.Create(domain.VehiclePublication{ID: "pending", SellerID: "seller-1", Status: domain.StatusPending})

	if w := sold(other, "a1"); w.Code != http.StatusForbidden {
		t.Errorf("other seller: expected 403, got %d", w.Code)
	}
	if p, _ := vehicleStore.FindByID("a1"); p.Status != domain.StatusApproved {
		t.Errorf("other seller must not change the status, got %s", p.Status)
	}

	if w := sold(owner, "a1"); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":"SOLD"`) {
		t.Errorf("owner: expected 200 SOLD, got %d %s", w.Code, w.Body.String())
	}
	if len(vehicleStore.FindApproved()) != 0 {
		t.Error("sold publication must leave the catalog")
	}

	if w := sold(owner, "a1"); w.Code != http.StatusConflict {
		t.Errorf("sold twice: expected 409, got %d", w.Code)
	}
	if w := sold(owner, "pending"); w.Code != http.StatusConflict {
		t.Errorf("pending: expected 409, got %d", w.Code)
	}
	if w := sold(owner, "nope"); w.Code != http.StatusNotFound {
		t.Errorf("missing: expected 404, got %d", w.Code)
	}
}
