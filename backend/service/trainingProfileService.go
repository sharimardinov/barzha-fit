package service

import (
	"context"

	"barzhafit/backend/domain"
)

type TrainingProfileStorage interface {
	Upsert(ctx context.Context, p domain.TrainingProfile) error
	Get(ctx context.Context, chatID int64) (domain.TrainingProfile, bool, error)
}

type TrainingProfileService struct {
	repo TrainingProfileStorage
}

func NewTrainingProfileService(repo TrainingProfileStorage) *TrainingProfileService {
	return &TrainingProfileService{repo: repo}
}

func (s *TrainingProfileService) Save(ctx context.Context, p domain.TrainingProfile) error {
	return s.repo.Upsert(ctx, p)
}

func (s *TrainingProfileService) Get(ctx context.Context, chatID int64) (domain.TrainingProfile, bool, error) {
	return s.repo.Get(ctx, chatID)
}
