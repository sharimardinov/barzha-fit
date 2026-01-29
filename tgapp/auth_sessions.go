package tgapp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
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
	secret  []byte
}

func newSessionStore(secret string) *sessionStore {
	return &sessionStore{
		byToken: make(map[string]authSession),
		secret:  []byte(secret),
	}
}

func (s *sessionStore) Create(userID int64, username string) authSession {
	now := time.Now()
	expiresAt := now.Add(authSessionTTL)
	if len(s.secret) == 0 {
		return s.createInMemory(userID, username, now, expiresAt)
	}
	payload := authSessionPayload{
		UserID:   userID,
		Username: username,
		Expires:  expiresAt.Unix(),
		IssuedAt: now.Unix(),
		Nonce:    randomToken(8),
	}
	token := signSessionPayload(payload, s.secret)
	return authSession{
		Token:     token,
		UserID:    userID,
		Username:  username,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
}

func (s *sessionStore) createInMemory(userID int64, username string, now time.Time, expiresAt time.Time) authSession {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		// Fallback to time-based token if RNG fails (should be extremely rare).
		token = []byte(time.Now().Format(time.RFC3339Nano))
	}
	session := authSession{
		Token:     hex.EncodeToString(token),
		UserID:    userID,
		Username:  username,
		CreatedAt: now,
		ExpiresAt: expiresAt,
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
	if len(s.secret) > 0 {
		return s.getSigned(token)
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

type authSessionPayload struct {
	UserID   int64  `json:"uid"`
	Username string `json:"usr,omitempty"`
	Expires  int64  `json:"exp"`
	IssuedAt int64  `json:"iat"`
	Nonce    string `json:"nonce"`
}

func (s *sessionStore) getSigned(token string) (authSession, bool) {
	payload, ok := verifySessionPayload(token, s.secret)
	if !ok {
		return authSession{}, false
	}
	exp := time.Unix(payload.Expires, 0)
	if time.Now().After(exp) {
		return authSession{}, false
	}
	issuedAt := time.Unix(payload.IssuedAt, 0)
	return authSession{
		Token:     token,
		UserID:    payload.UserID,
		Username:  payload.Username,
		CreatedAt: issuedAt,
		ExpiresAt: exp,
	}, true
}

func signSessionPayload(payload authSessionPayload, secret []byte) string {
	raw, _ := json.Marshal(payload)
	enc := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(enc))
	sig := hex.EncodeToString(mac.Sum(nil))
	return enc + "." + sig
}

func verifySessionPayload(token string, secret []byte) (authSessionPayload, bool) {
	parts := splitToken(token)
	if len(parts) != 2 {
		return authSessionPayload{}, false
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return authSessionPayload{}, false
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0]))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return authSessionPayload{}, false
	}
	var payload authSessionPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return authSessionPayload{}, false
	}
	if payload.UserID == 0 || payload.Expires == 0 {
		return authSessionPayload{}, false
	}
	return payload, true
}

func splitToken(token string) []string {
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			return []string{token[:i], token[i+1:]}
		}
	}
	return nil
}
