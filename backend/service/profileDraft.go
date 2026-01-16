package service

import (
	"errors"
	"sync"
)

var ErrDraftNotFound = errors.New("profile draft not found")

type ProfileDraft struct {
	ChatID        int64
	Sex           string
	HeightCM      int
	WeightKG      float64
	BodyFatPct    float64
	Age           int
	Activity      string
	Goal          string
	ActivityErr   error
	ActivityReady bool
	readyCh       chan struct{}
}

type ProfileDraftStore struct {
	mu sync.RWMutex
	m  map[int64]*ProfileDraft
}

func NewProfileDraftStore() *ProfileDraftStore {
	return &ProfileDraftStore{m: make(map[int64]*ProfileDraft)}
}

func (s *ProfileDraftStore) Start(chatID int64) *ProfileDraft {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := &ProfileDraft{
		ChatID:   chatID,
		Activity: "mid",
		Goal:     "balance",
		readyCh:  make(chan struct{}),
	}
	s.m[chatID] = d
	return d
}

func (s *ProfileDraftStore) Get(chatID int64) (*ProfileDraft, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.m[chatID]
	return d, ok
}

func (s *ProfileDraftStore) Snapshot(chatID int64) (ProfileDraft, <-chan struct{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.m[chatID]
	if !ok {
		return ProfileDraft{}, nil, false
	}
	cp := *d
	return cp, d.readyCh, true
}

func (s *ProfileDraftStore) Update(chatID int64, fn func(d *ProfileDraft)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.m[chatID]
	if !ok {
		return false
	}
	fn(d)
	return true
}

func (s *ProfileDraftStore) SetActivity(chatID int64, activity string, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.m[chatID]
	if !ok {
		return ErrDraftNotFound
	}
	if activity != "" {
		d.Activity = activity
	}
	d.ActivityErr = err
	if !d.ActivityReady {
		d.ActivityReady = true
		close(d.readyCh)
	}
	return nil
}

func (s *ProfileDraftStore) Clear(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, chatID)
}
