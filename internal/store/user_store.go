package store

import (
	"errors"
	"os"
	"sync"

	"automarket/internal/auth"
	"automarket/internal/domain"
)

type UserStore struct {
	mu    sync.RWMutex
	users map[string]domain.User // clave: ID de usuario
}

func NewUserStore() *UserStore {
	return &UserStore{
		users: make(map[string]domain.User),
	}
}

func (s *UserStore) Create(u domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.users {
		if existing.Username == u.Username {
			return errors.New("username already exists")
		}
	}

	s.users[u.ID] = u
	return nil
}

func (s *UserStore) FindByUsername(username string) (domain.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.Username == username {
			return u, true
		}
	}
	return domain.User{}, false
}

func (s *UserStore) FindByID(id string) (domain.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[id]
	return u, ok
}

func (s *UserStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[id]; !ok {
		return false
	}
	delete(s.users, id)
	return true
}

func (s *UserStore) SeedAdmin() error {
	username := os.Getenv("ADMIN_USERNAME")
	password := os.Getenv("ADMIN_PASSWORD")
	if username == "" || password == "" {
		return errors.New("missing required environment variables: ADMIN_USERNAME and ADMIN_PASSWORD must be defined")
	}

	if _, exists := s.FindByUsername(username); exists {
		return nil
	}

	salt := auth.GenerateSalt()
	hash := auth.HashPassword(password, salt)

	admin := domain.User{
		ID:           "admin-1",
		Username:     username,
		PasswordHash: hash,
		Salt:         salt,
		Role:         domain.RoleAdmin,
		ContactInfo:  "admin@automarket.com",
	}

	return s.Create(admin)
}

func SeedAdmin(store *UserStore) error {
	return store.SeedAdmin()
}


