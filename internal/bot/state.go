package bot

import (
	"sync"

	"barzhafit/internal/domain"
)

type StateStore struct {
	mu sync.RWMutex
	m  map[int64]domain.State
}

func NewStateStore() *StateStore {
	return &StateStore{m: make(map[int64]domain.State)}
}

func (s *StateStore) Get(chatID int64) domain.State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[chatID]
}

func (s *StateStore) Set(chatID int64, st domain.State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st == domain.StateNone {
		delete(s.m, chatID)
		return
	}
	s.m[chatID] = st
}

func (s *StateStore) Clear(chatID int64) {
	s.Set(chatID, domain.StateNone)
}
