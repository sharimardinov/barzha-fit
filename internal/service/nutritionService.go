package service

import (
	"barzhafit/internal/storage/db"
	"context"
	"strings"
	"time"
)

type NutritionService struct {
	meals *db.MealRepo
	ai    *AIService
}

func NewNutritionService(meals *db.MealRepo, ai *AIService) *NutritionService {
	return &NutritionService{meals: meals, ai: ai}
}

func (s *NutritionService) AddMealFromText(ctx context.Context, chatID int64, eatenAt time.Time, text string) (db.Meal, error) {
	text = strings.TrimSpace(text)

	est, raw, err := s.ai.EstimateNutrition(ctx, text)
	if err != nil {
		// даже при ошибке AI можем сохранить “черновик” как есть
		m := db.Meal{ChatID: chatID, EatenAt: eatenAt, Text: text}
		_ = s.meals.Add(ctx, m, map[string]any{"error": err.Error()})
		return m, err
	}

	m := db.Meal{
		ChatID:   chatID,
		EatenAt:  eatenAt,
		Text:     text,
		Kcal:     est.Kcal,
		ProteinG: est.ProteinG,
		FatG:     est.FatG,
		CarbsG:   est.CarbsG,
	}

	// ВОТ ЭТОГО У ТЕБЯ НЕ БЫЛО: сохраняем в БД
	if err := s.meals.Add(ctx, m, raw); err != nil {
		return m, err
	}

	return m, nil
}

func (s *NutritionService) UndoLast(ctx context.Context, chatID int64) (bool, error) {
	return s.meals.DeleteLast(ctx, chatID)
}

func (s *NutritionService) ListToday(ctx context.Context, chatID int64, loc *time.Location, now time.Time) ([]db.Meal, error) {
	from := dayStart(now, loc)
	to := from.Add(24 * time.Hour)
	return s.meals.ListByDay(ctx, chatID, from, to)
}

func (s *NutritionService) SumToday(ctx context.Context, chatID int64, loc *time.Location, now time.Time) (kcal, p, f, c int, err error) {
	from := dayStart(now, loc)
	to := from.Add(24 * time.Hour)
	return s.meals.SumByDay(ctx, chatID, from, to)
}

func dayStart(t time.Time, loc *time.Location) time.Time {
	tt := t.In(loc)
	return time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, loc)
}
