package service

import (
	"context"
	"fmt"
)

type StepsStorage interface {
	Upsert(ctx context.Context, userID int64, dayDate string, steps int) error
	Get(ctx context.Context, userID int64, dayDate string) (int, bool, error)
	ListByRange(ctx context.Context, userID int64, fromDate, toDate string) (map[string]int, error)
}

type StepsService struct {
	steps StepsStorage
}

func NewStepsService(steps StepsStorage) *StepsService { return &StepsService{steps: steps} }

func (s *StepsService) SetSteps(ctx context.Context, userID int64, dayDate string, steps int) error {
	if steps < 0 {
		return fmt.Errorf("steps must be >= 0")
	}
	return s.steps.Upsert(ctx, userID, dayDate, steps)
}

func (s *StepsService) GetByDate(ctx context.Context, userID int64, dayDate string) (int, bool, error) {
	return s.steps.Get(ctx, userID, dayDate)
}

func (s *StepsService) ListByRange(ctx context.Context, userID int64, fromDate, toDate string) (map[string]int, error) {
	return s.steps.ListByRange(ctx, userID, fromDate, toDate)
}
