package store

import (
	"sync"

	"automarket/internal/domain"
)

type VehicleStore struct {
	mu   sync.RWMutex
	pubs map[string]domain.VehiclePublication
}

func NewVehicleStore() *VehicleStore {
	return &VehicleStore{
		pubs: make(map[string]domain.VehiclePublication),
	}
}
