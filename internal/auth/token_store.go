package auth

import "sync"

type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]string // token -> userID
}

func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens: make(map[string]string),
	}
}

func (s *TokenStore) CreateToken(userID string) string {
	token := GenerateToken()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = userID
	return token
}

func (s *TokenStore) Set(token, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = userID
}

func (s *TokenStore) GetUserID(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userID, ok := s.tokens[token]
	return userID, ok
}

func (s *TokenStore) DeleteToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
}

func (s *TokenStore) DeleteByUserID(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for t, uid := range s.tokens {
		if uid == userID {
			delete(s.tokens, t)
		}
	}
}
