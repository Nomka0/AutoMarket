package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"automarket/internal/auth"
	"automarket/internal/domain"
)

type UserStore struct {
	mu       sync.RWMutex
	users    map[string]domain.User // clave: ID de usuario
	filePath string
}

func NewUserStore(filePath ...string) *UserStore {
	s := &UserStore{
		users: make(map[string]domain.User),
	}
	if len(filePath) > 0 && filePath[0] != "" {
		s.filePath = filePath[0]
		_ = s.load()
	}
	return s
}

func (s *UserStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var loaded map[string]domain.User
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	if loaded != nil {
		s.users = loaded
	}
	return nil
}

func (s *UserStore) save() error {
	if s.filePath == "" {
		return nil
	}
	if dir := filepath.Dir(s.filePath); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
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
	_ = s.save()
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
	_ = s.save()
	return true
}

func (s *UserStore) FindAll() []domain.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]domain.User, 0, len(s.users))
	for _, u := range s.users {
		res = append(res, u)
	}
	return res
}

func (s *UserStore) HasAdmin() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.Role == domain.RoleAdmin {
			return true
		}
	}
	return false
}

func (s *UserStore) CreateAdmin(username, password, contactInfo string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, u := range s.users {
		if u.Role == domain.RoleAdmin {
			return domain.User{}, errors.New("admin user already exists")
		}
		if u.Username == username {
			return domain.User{}, errors.New("username already exists")
		}
	}

	salt := auth.GenerateSalt()
	admin := domain.User{
		ID:           "admin-1",
		Username:     username,
		PasswordHash: auth.HashPassword(password, salt),
		Salt:         salt,
		Role:         domain.RoleAdmin,
		ContactInfo:  contactInfo,
	}

	s.users[admin.ID] = admin
	_ = s.save()
	return admin, nil
}

func (s *UserStore) SeedAdmin() error {
	username := os.Getenv("ADMIN_USERNAME")
	password := os.Getenv("ADMIN_PASSWORD")
	if username == "" || password == "" {
		return nil
	}

	if s.HasAdmin() {
		return nil
	}

	_, err := s.CreateAdmin(username, password, "admin@automarket.com")
	return err
}

func SeedAdmin(store *UserStore) error {
	return store.SeedAdmin()
}


