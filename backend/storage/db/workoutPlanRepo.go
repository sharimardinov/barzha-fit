package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkoutPlanRepo struct {
	db *pgxpool.Pool
}

func NewWorkoutPlanRepo(db *pgxpool.Pool) *WorkoutPlanRepo { return &WorkoutPlanRepo{db: db} }

func (r *WorkoutPlanRepo) Get(ctx context.Context, chatID int64) (int64, []byte, bool, error) {
	var id int64
	var payload []byte
	err := r.db.QueryRow(ctx, `
		select id, payload
		from workout_plans
		where chat_id=$1
	`, chatID).Scan(&id, &payload)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil, false, nil
		}
		return 0, nil, false, err
	}
	return id, payload, true, nil
}

func (r *WorkoutPlanRepo) Upsert(ctx context.Context, chatID int64, payload []byte) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		insert into workout_plans(chat_id, payload)
		values ($1, $2::jsonb)
		on conflict (chat_id)
		do update set payload=excluded.payload, updated_at=now()
		returning id
	`, chatID, string(payload)).Scan(&id)
	return id, err
}
