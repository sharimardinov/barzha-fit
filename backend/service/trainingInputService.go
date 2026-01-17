package service

import (
	"context"
	"fmt"
	"strings"

	"barzhafit/backend/domain"
)

type TrainingInputStorage interface {
	Upsert(ctx context.Context, input domain.TrainingInput) (domain.TrainingInput, error)
}

type UserIdentityStorage interface {
	EnsureByTelegramChatID(ctx context.Context, chatID int64) (string, error)
}

type TrainingInputService struct {
	repo  TrainingInputStorage
	users UserIdentityStorage
}

func NewTrainingInputService(repo TrainingInputStorage, users UserIdentityStorage) *TrainingInputService {
	return &TrainingInputService{repo: repo, users: users}
}

func (s *TrainingInputService) SaveFromSelection(ctx context.Context, chatID int64, levelRaw, goalRaw string, daysPerWeek int, injuries []string) (domain.TrainingInput, error) {
	if chatID <= 0 {
		return domain.TrainingInput{}, fmt.Errorf("invalid chat_id")
	}
	level, ok := domain.NormalizeFitnessLevel(levelRaw)
	if !ok {
		return domain.TrainingInput{}, fmt.Errorf("invalid fitness_level")
	}
	goal, ok := domain.NormalizeFitnessGoal(goalRaw)
	if !ok {
		return domain.TrainingInput{}, fmt.Errorf("invalid goal")
	}
	if daysPerWeek < 2 || daysPerWeek > 6 {
		return domain.TrainingInput{}, fmt.Errorf("invalid days_per_week")
	}

	userID, err := s.users.EnsureByTelegramChatID(ctx, chatID)
	if err != nil {
		return domain.TrainingInput{}, err
	}

	input := domain.TrainingInput{
		UserID:      userID,
		Level:       level,
		Goal:        goal,
		DaysPerWeek: daysPerWeek,
		Injuries:    normalizeInjuries(injuries),
	}
	return s.repo.Upsert(ctx, input)
}

func normalizeInjuries(injuries []string) []string {
	if len(injuries) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(injuries))
	for _, item := range injuries {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		out = append(out, strings.ToLower(value))
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}
