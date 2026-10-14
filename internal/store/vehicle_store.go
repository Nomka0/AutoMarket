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

func (s *VehicleStore) Create(pub domain.VehiclePublication) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pubs[pub.ID] = pub
}

func (s *VehicleStore) FindByID(id string) (domain.VehiclePublication, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pub, ok := s.pubs[id]
	return pub, ok
}

func (s *VehicleStore) FindApproved() []domain.VehiclePublication {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := []domain.VehiclePublication{} // lista vacia se serializa como [] y no como null
	for _, pub := range s.pubs {
		if pub.Status == domain.StatusApproved {
			res = append(res, pub)
		}
	}
	return res
}

func (s *VehicleStore) FindPending() []domain.VehiclePublication {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := []domain.VehiclePublication{} // lista vacia se serializa como [] y no como null
	for _, pub := range s.pubs {
		if pub.Status == domain.StatusPending {
			res = append(res, pub)
		}
	}
	return res
}

func (s *VehicleStore) UpdateStatus(id, newStatus string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	pub, ok := s.pubs[id]
	if !ok {
		return false
	}
	pub.Status = newStatus
	s.pubs[id] = pub
	return true
}

func (s *VehicleStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pubs[id]; !ok {
		return false
	}
	delete(s.pubs, id)
	return true
}

