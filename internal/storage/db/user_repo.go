package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Ensure(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, `
		insert into users (user_id) values ($1)
		on conflict (user_id) do nothing
	`, userID)
	return err
}

func (r *UserRepo) GetCycleDay(ctx context.Context, userID int64) (int, error) {
	var day int
	err := r.db.QueryRow(ctx, `select cycle_day from users where user_id=$1`, userID).Scan(&day)
	return day, err
}

func (r *UserRepo) SetCycleDay(ctx context.Context, userID int64, day int) error {
	_, err := r.db.Exec(ctx, `
		update users set cycle_day=$2, updated_at=now()
		where user_id=$1
	`, userID, day)
	return err
}
