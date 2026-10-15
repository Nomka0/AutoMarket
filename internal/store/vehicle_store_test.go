package store_test

import (
	"sync"
	"testing"
	"time"

	"automarket/internal/domain"
	"automarket/internal/store"
)

func TestVehicleStore_CRUD(t *testing.T) {
	s := store.NewVehicleStore()

	pub := domain.VehiclePublication{
		ID:            "v-1",
		SellerID:      "u-1",
		SellerContact: "test@example.com",
		Title:         "Toyota Corolla",
		Description:   "Buen estado",
		Price:         15000,
		Year:          2020,
		Status:        domain.StatusPending,
		CreatedAt:     time.Now(),
	}

	s.Create(pub)

	found, ok := s.FindByID("v-1")
	if !ok || found.Title != "Toyota Corolla" {
		t.Fatalf("expected vehicle found, got ok=%v, found=%+v", ok, found)
	}

	pending := s.FindPending()
	if len(pending) != 1 || pending[0].ID != "v-1" {
		t.Fatalf("expected 1 pending publication, got %d", len(pending))
	}

	approved := s.FindApproved()
	if len(approved) != 0 {
		t.Fatalf("expected 0 approved publications, got %d", len(approved))
	}

	if !s.UpdateStatus("v-1", domain.StatusApproved) {
		t.Fatal("expected update status to succeed")
	}

	if s.UpdateStatus("non-existent", domain.StatusApproved) {
		t.Fatal("expected update status on non-existent to fail")
	}

	approved = s.FindApproved()
	if len(approved) != 1 || approved[0].ID != "v-1" {
		t.Fatalf("expected 1 approved publication, got %d", len(approved))
	}

	if !s.Delete("v-1") {
		t.Fatal("expected delete to succeed")
	}

	if s.Delete("v-1") {
		t.Fatal("expected subsequent delete to fail")
	}

	_, ok = s.FindByID("v-1")
	if ok {
		t.Fatal("expected vehicle to be deleted")
	}
}

func TestVehicleStore_Concurrency(t *testing.T) {
	s := store.NewVehicleStore()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := "v"
			s.Create(domain.VehiclePublication{
				ID:     id,
				Status: domain.StatusPending,
			})
			s.FindByID(id)
			s.UpdateStatus(id, domain.StatusApproved)
			s.FindApproved()
			s.FindPending()
		}(i)
	}

	wg.Wait()
}

func TestVehicleStore_JSONPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := tmpDir + "/vehicles_test.json"

	s1 := store.NewVehicleStore(filePath)
	pub := domain.VehiclePublication{
		ID:            "v-persist-1",
		SellerID:      "seller-1",
		SellerContact: "seller@test.com",
		Title:         "Mazda CX-30",
		Description:   "Excelente estado",
		Price:         85000000,
		Year:          2022,
		Status:        domain.StatusPending,
		CreatedAt:     time.Now().UTC(),
	}
	s1.Create(pub)

	// Simular reinicio
	s2 := store.NewVehicleStore(filePath)
	found, ok := s2.FindByID("v-persist-1")
	if !ok || found.Title != "Mazda CX-30" {
		t.Fatalf("expected persisted vehicle, got ok=%v, pub=%+v", ok, found)
	}
	if len(s2.FindPending()) != 1 {
		t.Fatalf("expected 1 pending vehicle, got %d", len(s2.FindPending()))
	}

	// Actualizar estado y verificar persistencia
	if !s2.UpdateStatus("v-persist-1", domain.StatusApproved) {
		t.Fatal("expected update status to succeed")
	}

	s3 := store.NewVehicleStore(filePath)
	approved := s3.FindApproved()
	if len(approved) != 1 || approved[0].Status != domain.StatusApproved {
		t.Fatalf("expected 1 approved vehicle in reloaded store, got %d", len(approved))
	}

	// Eliminar y verificar persistencia
	if !s3.Delete("v-persist-1") {
		t.Fatal("expected delete to succeed")
	}

	s4 := store.NewVehicleStore(filePath)
	if _, ok := s4.FindByID("v-persist-1"); ok {
		t.Fatal("expected deleted vehicle to remain deleted after reload")
	}
}

