package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"barzhafit/backend/storage/db"
)

var (
	ErrNutritionAI   = errors.New("nutrition: ai_failed")
	ErrNutritionSave = errors.New("nutrition: save_failed")
)

type NutritionService struct {
	meals MealStorage
	ai    *NutritionAI
}

type MealStorage interface {
	Add(ctx context.Context, m *db.Meal, aiRaw any) error
	DeleteLast(ctx context.Context, chatID int64) (bool, error)
	DeleteByID(ctx context.Context, chatID int64, id int64) (bool, error)
	ListByDay(ctx context.Context, chatID int64, from, to time.Time) ([]db.Meal, error)
	ListRecent(ctx context.Context, chatID int64, limit int) ([]db.Meal, error)
	SumByDay(ctx context.Context, chatID int64, from, to time.Time) (kcal, p, f, c int, err error)
	SumAllTime(ctx context.Context, chatID int64) (kcal, p, f, c int, err error)
	SumByRangeDaily(ctx context.Context, chatID int64, from, to time.Time, tz string) (map[string]db.DayNutrition, error)
}

func NewNutritionService(meals MealStorage, ai *NutritionAI) *NutritionService {
	return &NutritionService{meals: meals, ai: ai}
}

func (s *NutritionService) AddMealFromText(ctx context.Context, chatID int64, eatenAt time.Time, text string) (db.Meal, error) {
	text = strings.TrimSpace(text)

	est, raw, err := s.ai.EstimateNutrition(ctx, text)

	// сформируем meal в любом случае
	m := db.Meal{
		ChatID:  chatID,
		EatenAt: eatenAt,
		Text:    text,
	}

	// если AI упал — сохраняем хотя бы текст + ai_raw с ошибкой
	if err != nil {
		if addErr := s.meals.Add(ctx, &m, map[string]any{"error": err.Error()}); addErr != nil {
			return m, fmt.Errorf("%w: %v", ErrNutritionSave, addErr)
		}
		return m, fmt.Errorf("%w: %v", ErrNutritionAI, err)
	}

	// если AI ок — заполняем КБЖУ и сохраняем
	m.ProteinG = est.ProteinG
	m.FatG = est.FatG
	m.CarbsG = est.CarbsG
	m.Kcal = m.ProteinG*4 + m.FatG*9 + m.CarbsG*4

	if err := s.meals.Add(ctx, &m, raw); err != nil {
		return m, fmt.Errorf("%w: %v", ErrNutritionSave, err)
	}

	return m, nil
}

func (s *NutritionService) UndoLast(ctx context.Context, chatID int64) (bool, error) {
	return s.meals.DeleteLast(ctx, chatID)
}

func (s *NutritionService) DeleteByID(ctx context.Context, chatID int64, id int64) (bool, error) {
	return s.meals.DeleteByID(ctx, chatID, id)
}

func (s *NutritionService) ListToday(ctx context.Context, chatID int64, loc *time.Location, now time.Time) ([]db.Meal, error) {
	from := dayStart(now, loc)
	to := from.Add(24 * time.Hour)
	return s.meals.ListByDay(ctx, chatID, from, to)
}

func (s *NutritionService) ListRecent(ctx context.Context, chatID int64, limit int) ([]db.Meal, error) {
	return s.meals.ListRecent(ctx, chatID, limit)
}

func (s *NutritionService) SumToday(ctx context.Context, chatID int64, loc *time.Location, now time.Time) (kcal, p, f, c int, err error) {
	from := dayStart(now, loc)
	to := from.Add(24 * time.Hour)
	return s.meals.SumByDay(ctx, chatID, from, to)
}

func (s *NutritionService) SumByDay(ctx context.Context, chatID int64, from, to time.Time) (kcal, p, f, c int, err error) {
	return s.meals.SumByDay(ctx, chatID, from, to)
}

func (s *NutritionService) SumByWeek(ctx context.Context, chatID int64, from, to time.Time, tz string) (kcal, p, f, c int, err error) {
	dailyMap, err := s.meals.SumByRangeDaily(ctx, chatID, from, to, tz)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	for _, dn := range dailyMap {
		kcal += dn.Kcal
		p += dn.P
		f += dn.F
		c += dn.C
	}
	return kcal, p, f, c, nil
}

func (s *NutritionService) SumByRangeDaily(ctx context.Context, chatID int64, from, to time.Time, tz string) (map[string]db.DayNutrition, error) {
	return s.meals.SumByRangeDaily(ctx, chatID, from, to, tz)
}

func (s *NutritionService) SumAllTime(ctx context.Context, chatID int64) (kcal, p, f, c int, err error) {
	return s.meals.SumAllTime(ctx, chatID)
}

func dayStart(t time.Time, loc *time.Location) time.Time {
	tt := t.In(loc)
	return time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, loc)
}
