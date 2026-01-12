package service

import (
	"context"
	"fmt"
)

type WorkoutStorage interface {
	UpsertStatus(ctx context.Context, userID int64, dayDate string, cycleDay int, status string) error
	GetStatus(ctx context.Context, userID int64, dayDate string) (string, bool, error)
}

type CycleStorage interface {
	Ensure(ctx context.Context, userID int64) error
	GetCycleDay(ctx context.Context, userID int64) (int, error)
	SetCycleDay(ctx context.Context, userID int64, day int) error
}

type WorkoutService struct {
	workout WorkoutStorage
	users   CycleStorage
}

func NewWorkoutService(workout WorkoutStorage, users CycleStorage) *WorkoutService {
	return &WorkoutService{workout: workout, users: users}
}

func (s *WorkoutService) GetCycleDay(ctx context.Context, userID int64) (int, error) {
	if err := s.users.Ensure(ctx, userID); err != nil {
		return 0, err
	}
	day, err := s.users.GetCycleDay(ctx, userID)
	if err != nil {
		return 0, err
	}
	if day < 1 || day > 7 {
		day = 1
		_ = s.users.SetCycleDay(ctx, userID, day)
	}
	return day, nil
}

func (s *WorkoutService) GetStatusByDate(ctx context.Context, userID int64, dayDate string) (string, bool, error) {
	return s.workout.GetStatus(ctx, userID, dayDate)
}

func (s *WorkoutService) MarkAndAdvance(ctx context.Context, userID int64, dayDate string, status string) (int, error) {
	if status != "done" && status != "skip" {
		return 0, fmt.Errorf("bad status=%q", status)
	}

	cycleDay, err := s.GetCycleDay(ctx, userID)
	if err != nil {
		return 0, err
	}

	if err := s.workout.UpsertStatus(ctx, userID, dayDate, cycleDay, status); err != nil {
		return 0, err
	}

	next := cycleDay + 1
	if next > 7 {
		next = 1
	}
	if err := s.users.SetCycleDay(ctx, userID, next); err != nil {
		return 0, err
	}

	return cycleDay, nil
}
