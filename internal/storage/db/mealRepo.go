package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Meal struct {
	ID       int64
	ChatID   int64
	EatenAt  time.Time
	Text     string
	Calories int
}

type MealRepo struct{ db *pgxpool.Pool }

func NewMealRepo(db *pgxpool.Pool) *MealRepo { return &MealRepo{db: db} }

func (r *MealRepo) Add(ctx context.Context, chatID int64, eatenAt time.Time, text string, calories int) error {
	_, err := r.db.Exec(ctx, `
		insert into meals(chat_id, eaten_at, text, calories)
		values ($1, $2, $3, $4)
	`, chatID, eatenAt, text, calories)
	return err
}

func (r *MealRepo) ListByDay(ctx context.Context, chatID int64, dayStart, dayEnd time.Time) ([]Meal, error) {
	rows, err := r.db.Query(ctx, `
		select id, chat_id, eaten_at, text, calories
		from meals
		where chat_id=$1 and eaten_at >= $2 and eaten_at < $3
		order by eaten_at asc
	`, chatID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Meal
	for rows.Next() {
		var m Meal
		if err := rows.Scan(&m.ID, &m.ChatID, &m.EatenAt, &m.Text, &m.Calories); err != nil {
			return nil, err
		}
		res = append(res, m)
	}
	return res, rows.Err()
}

func (r *MealRepo) SumCaloriesByDay(ctx context.Context, chatID int64, dayStart, dayEnd time.Time) (int, error) {
	var sum int
	err := r.db.QueryRow(ctx, `
		select coalesce(sum(calories), 0)
		from meals
		where chat_id=$1 and eaten_at >= $2 and eaten_at < $3
	`, chatID, dayStart, dayEnd).Scan(&sum)
	return sum, err
}

func (r *MealRepo) DeleteLast(ctx context.Context, chatID int64) (bool, error) {
	ct, err := r.db.Exec(ctx, `
		delete from meals
		where id = (
			select id from meals where chat_id=$1 order by eaten_at desc, id desc limit 1
		)
	`, chatID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}
