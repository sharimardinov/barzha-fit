package service

import (
	"barzhafit/backend/domain"
	"barzhafit/backend/util"
	"context"
)

type ProfileStorage interface {
	Upsert(ctx context.Context, p domain.Profile) error
	Get(ctx context.Context, chatID int64) (domain.Profile, bool, error)
}

type ProfileService struct {
	repo ProfileStorage
}

func NewProfileService(repo ProfileStorage) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) SaveFromText(ctx context.Context, chatID int64, text string) (domain.Profile, error) {
	p := util.ParseProfileText(chatID, text)
	return p, s.repo.Upsert(ctx, p)
}

func (s *ProfileService) Save(ctx context.Context, p domain.Profile) error {
	return s.repo.Upsert(ctx, p)
}

func (s *ProfileService) Get(ctx context.Context, chatID int64) (domain.Profile, bool, error) {
	return s.repo.Get(ctx, chatID)
}

func (s *ProfileService) UpdateWeight(ctx context.Context, chatID int64, weightKG float64) (domain.Profile, bool, error) {
	p, ok, err := s.repo.Get(ctx, chatID)
	if err != nil || !ok {
		return domain.Profile{}, ok, err
	}
	p.WeightKG = weightKG
	if err := s.repo.Upsert(ctx, p); err != nil {
		return domain.Profile{}, true, err
	}
	return p, true, nil
}
