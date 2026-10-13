package auth

import (
	"sync"
	"time"
)

const sessionTTL = 24 * time.Hour

type Session struct {
	UserID    string
	ExpiresAt time.Time
}

type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]Session
}

func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens: make(map[string]Session),
	}
}

func (s *TokenStore) CreateToken(userID string) string {
	token := GenerateToken()
	s.Set(token, userID)
	return token
}

func (s *TokenStore) Set(token, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = Session{UserID: userID, ExpiresAt: time.Now().Add(sessionTTL)}
}

// honey: los tokens expirados no se purgan del mapa; añadir limpieza periódica si el proceso vive días con muchos logins.
func (s *TokenStore) ValidateToken(token string) (Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.tokens[token]
	if !ok || time.Now().After(session.ExpiresAt) {
		return Session{}, false
	}
	return session, true
}

func (s *TokenStore) GetUserID(token string) (string, bool) {
	session, ok := s.ValidateToken(token)
	return session.UserID, ok
}

func (s *TokenStore) DeleteToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
}

func (s *TokenStore) DeleteByUserID(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for t, session := range s.tokens {
		if session.UserID == userID {
			delete(s.tokens, t)
		}
	}
}
