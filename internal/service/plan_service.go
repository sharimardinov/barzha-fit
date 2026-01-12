package service

import "context"

type PlanStorage interface {
	Save(ctx context.Context, userID int64, text string) error
	Get(ctx context.Context, userID int64) (string, error)
}

type PlanService struct {
	repo PlanStorage
}

func NewPlanService(repo PlanStorage) *PlanService {
	return &PlanService{repo: repo}
}

func (s *PlanService) Save(ctx context.Context, userID int64, text string) error {
	return s.repo.Save(ctx, userID, text)
}

func (s *PlanService) Get(ctx context.Context, userID int64) (string, error) {
	return s.repo.Get(ctx, userID)
}
