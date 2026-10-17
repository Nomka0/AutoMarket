package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"automarket/internal/domain"
)

type VehicleStore struct {
	mu       sync.RWMutex
	pubs     map[string]domain.VehiclePublication
	filePath string
}

func NewVehicleStore(filePath ...string) *VehicleStore {
	s := &VehicleStore{
		pubs: make(map[string]domain.VehiclePublication),
	}
	if len(filePath) > 0 && filePath[0] != "" {
		s.filePath = filePath[0]
		_ = s.load()
	}
	return s
}

func (s *VehicleStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var loaded map[string]domain.VehiclePublication
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	if loaded != nil {
		s.pubs = loaded
	}
	return nil
}

func (s *VehicleStore) save() error {
	if s.filePath == "" {
		return nil
	}
	if dir := filepath.Dir(s.filePath); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	data, err := json.MarshalIndent(s.pubs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *VehicleStore) Create(pub domain.VehiclePublication) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pubs[pub.ID] = pub
	_ = s.save()
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

func (s *VehicleStore) FindBySellerID(sellerID string) []domain.VehiclePublication {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := []domain.VehiclePublication{}
	for _, pub := range s.pubs {
		if pub.SellerID == sellerID {
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
	_ = s.save()
	return true
}

func (s *VehicleStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pubs[id]; !ok {
		return false
	}
	delete(s.pubs, id)
	_ = s.save()
	return true
}

func (s *VehicleStore) UpdateSellerContact(sellerID, sellerContact, sellerEmail, sellerPhone string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, pub := range s.pubs {
		if pub.SellerID == sellerID {
			pub.SellerContact = sellerContact
			pub.SellerEmail = sellerEmail
			pub.SellerPhone = sellerPhone
			s.pubs[id] = pub
		}
	}
	_ = s.save()
}

