package service

import (
	"context"
	"time"
)

type WorkoutStorage interface {
	UpsertStatus(ctx context.Context, userID int64, date time.Time, day int, status string) error
	GetStatus(ctx context.Context, userID int64, date time.Time) (string, bool, error)
}

type WorkoutService struct {
	workout WorkoutStorage
}

func NewWorkoutService(workout WorkoutStorage) *WorkoutService {
	return &WorkoutService{workout: workout}
}

func (s *WorkoutService) GetStatusToday(ctx context.Context, userID int64, now time.Time) (string, bool, error) {
	return s.workout.GetStatus(ctx, userID, now)
}

func (s *WorkoutService) MarkToday(ctx context.Context, userID int64, now time.Time, day int, status string) error {
	return s.workout.UpsertStatus(ctx, userID, now, day, status)
}
