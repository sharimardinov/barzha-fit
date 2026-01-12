package service

import (
	"context"
	"time"
)

type MealStorage interface {
	Add(ctx context.Context, chatID int64, eatenAt time.Time, text string, calories int) error
	ListByDay(ctx context.Context, chatID int64, dayStart, dayEnd time.Time) ([]any, error) // не будем использовать, ниже проще напрямую repo
	SumCaloriesByDay(ctx context.Context, chatID int64, dayStart, dayEnd time.Time) (int, error)
	DeleteLast(ctx context.Context, chatID int64) (bool, error)
}
