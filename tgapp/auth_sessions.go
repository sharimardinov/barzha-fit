package tgapp

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const authSessionTTL = 30 * 24 * time.Hour

type authSession struct {
	Token     string
	UserID    int64
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type sessionStore struct {
	mu      sync.RWMutex
	byToken map[string]authSession
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		byToken: make(map[string]authSession),
	}
}

func (s *sessionStore) Create(userID int64, username string) authSession {
	now := time.Now()
	session := authSession{
		Token:     uuid.NewString(),
		UserID:    userID,
		Username:  username,
		CreatedAt: now,
		ExpiresAt: now.Add(authSessionTTL),
	}
	s.mu.Lock()
	s.byToken[session.Token] = session
	s.mu.Unlock()
	return session
}

func (s *sessionStore) Get(token string) (authSession, bool) {
	if token == "" {
		return authSession{}, false
	}
	s.mu.RLock()
	session, ok := s.byToken[token]
	s.mu.RUnlock()
	if !ok {
		return authSession{}, false
	}
	if time.Now().After(session.ExpiresAt) {
		s.mu.Lock()
		delete(s.byToken, token)
		s.mu.Unlock()
		return authSession{}, false
	}
	return session, true
}
