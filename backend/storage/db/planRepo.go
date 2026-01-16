package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PlanRepo struct {
	db *pgxpool.Pool
}

func NewPlanRepo(db *pgxpool.Pool) *PlanRepo {
	return &PlanRepo{db: db}
}

func (r *PlanRepo) Save(ctx context.Context, userID int64, text string) error {
	_, err := r.db.Exec(ctx, `
		insert into plans (user_id, text)
		values ($1, $2)
		on conflict (user_id)
		do update set text = excluded.text, created_at = now()
	`, userID, text)
	return err
}

func (r *PlanRepo) Get(ctx context.Context, userID int64) (string, error) {
	var text string
	err := r.db.QueryRow(ctx, `select text from plans where user_id=$1`, userID).Scan(&text)
	return text, err
}
