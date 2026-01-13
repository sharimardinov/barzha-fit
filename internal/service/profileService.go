package service

import (
	"barzhafit/internal/domain"
	"barzhafit/internal/util"
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
