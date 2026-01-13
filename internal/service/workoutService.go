package service

import (
	"barzhafit/internal/util"
	"context"
	"fmt"
	"time"
)

type WorkoutStorage interface {
	UpsertStatus(ctx context.Context, userID int64, dayDate string, cycleDay int, status string) error
	GetStatus(ctx context.Context, userID int64, dayDate string) (string, bool, error)
	ListByRange(ctx context.Context, userID int64, fromDate, toDate string) (map[string]string, error)
}

type WorkoutService struct {
	workout WorkoutStorage
}

func NewWorkoutService(workout WorkoutStorage) *WorkoutService {
	return &WorkoutService{workout: workout}
}

func (s *WorkoutService) GetStatusByDate(ctx context.Context, userID int64, dayDate string) (string, bool, error) {
	return s.workout.GetStatus(ctx, userID, dayDate)
}

func (s *WorkoutService) ListByRange(ctx context.Context, userID int64, fromDate, toDate string) (map[string]string, error) {
	return s.workout.ListByRange(ctx, userID, fromDate, toDate)
}

func (s *WorkoutService) MarkAndAdvance(ctx context.Context, userID int64, dayDate string, status string) (int, error) {
	if status != "done" && status != "skip" {
		return 0, fmt.Errorf("bad status=%q", status)
	}

	cycleDay, err := cycleDayFromDate(dayDate)
	if err != nil {
		return 0, err
	}

	if err := s.workout.UpsertStatus(ctx, userID, dayDate, cycleDay, status); err != nil {
		return 0, err
	}

	return cycleDay, nil
}

func cycleDayFromDate(dayDate string) (int, error) {
	t, err := time.Parse("2006-01-02", dayDate)
	if err != nil {
		return 0, err
	}
	return util.Weekday1to7(t), nil
}
