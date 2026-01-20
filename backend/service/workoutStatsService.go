package service

import (
	"context"

	"barzhafit/backend/domain"
)

type StrengthStatsStorage interface {
	StrengthTotals(ctx context.Context, chatID int64) (domain.StrengthTotals, error)
	StrengthTopByTonnage(ctx context.Context, chatID int64, limit int) ([]domain.StrengthExerciseStats, error)
	StrengthTopByReps(ctx context.Context, chatID int64, limit int) ([]domain.StrengthExerciseStats, error)
	StrengthRecentEntries(ctx context.Context, chatID int64, exerciseName string, limit int) ([]domain.StrengthExerciseEntry, error)
}

type WorkoutStatsService struct {
	storage StrengthStatsStorage
}

func NewWorkoutStatsService(storage StrengthStatsStorage) *WorkoutStatsService {
	return &WorkoutStatsService{storage: storage}
}

func (s *WorkoutStatsService) StrengthAllTime(ctx context.Context, chatID int64) (domain.StrengthStats, error) {
	const (
		topLimit    = 5
		recentLimit = 4
		recentCount = 3
	)

	totals, err := s.storage.StrengthTotals(ctx, chatID)
	if err != nil {
		return domain.StrengthStats{}, err
	}
	topByTonnage, err := s.storage.StrengthTopByTonnage(ctx, chatID, topLimit)
	if err != nil {
		return domain.StrengthStats{}, err
	}
	topByReps, err := s.storage.StrengthTopByReps(ctx, chatID, topLimit)
	if err != nil {
		return domain.StrengthStats{}, err
	}

	base := topByTonnage
	if len(base) == 0 {
		base = topByReps
	}
	if len(base) > recentCount {
		base = base[:recentCount]
	}
	recent := make([]domain.StrengthExerciseRecent, 0, len(base))
	for _, item := range base {
		entries, err := s.storage.StrengthRecentEntries(ctx, chatID, item.Name, recentLimit)
		if err != nil {
			return domain.StrengthStats{}, err
		}
		recent = append(recent, domain.StrengthExerciseRecent{
			Name:    item.Name,
			Entries: entries,
		})
	}

	return domain.StrengthStats{
		Totals:       totals,
		TopByTonnage: topByTonnage,
		TopByReps:    topByReps,
		Recent:       recent,
	}, nil
}
