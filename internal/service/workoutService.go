package service

import (
	"context"
	"time"
)

type UserStorage interface {
	Ensure(ctx context.Context, userID int64) error
	GetCycleDay(ctx context.Context, userID int64) (int, error)
	SetCycleDay(ctx context.Context, userID int64, day int) error
}

type WorkoutStorage interface {
	UpsertStatus(ctx context.Context, userID int64, date time.Time, cycleDay int, status string) error
	GetStatus(ctx context.Context, userID int64, date time.Time) (string, bool, error)
	GetRecord(ctx context.Context, userID int64, date time.Time) (cycleDay int, status string, ok bool, err error)
}

type WorkoutService struct {
	users   UserStorage
	workout WorkoutStorage
}

func NewWorkoutService(users UserStorage, workout WorkoutStorage) *WorkoutService {
	return &WorkoutService{users: users, workout: workout}
}

func (s *WorkoutService) GetToday(ctx context.Context, userID int64, now time.Time) (cycleDay int, status string, has bool, err error) {
	if err := s.users.Ensure(ctx, userID); err != nil {
		return 0, "", false, err
	}
	day, err := s.users.GetCycleDay(ctx, userID)
	if err != nil {
		return 0, "", false, err
	}
	st, ok, err := s.workout.GetStatus(ctx, userID, now)
	if err != nil {
		return 0, "", false, err
	}
	return day, st, ok, nil
}

func (s *WorkoutService) MarkToday(ctx context.Context, userID int64, now time.Time, status string) (newCycleDay int, advanced bool, err error) {
	if err := s.users.Ensure(ctx, userID); err != nil {
		return 0, false, err
	}

	// если сегодня уже есть запись — просто меняем статус, день НЕ двигаем
	recDay, _, ok, err := s.workout.GetRecord(ctx, userID, now)
	if err != nil {
		return 0, false, err
	}
	if ok {
		if err := s.workout.UpsertStatus(ctx, userID, now, recDay, status); err != nil {
			return 0, false, err
		}
		current, err := s.users.GetCycleDay(ctx, userID)
		if err != nil {
			return 0, false, err
		}
		return current, false, nil
	}

	// иначе — фиксируем и двигаем цикл
	day, err := s.users.GetCycleDay(ctx, userID)
	if err != nil {
		return 0, false, err
	}

	if err := s.workout.UpsertStatus(ctx, userID, now, day, status); err != nil {
		return 0, false, err
	}

	next := day + 1
	if next > 7 {
		next = 1
	}
	if err := s.users.SetCycleDay(ctx, userID, next); err != nil {
		return 0, false, err
	}
	return next, true, nil
}
