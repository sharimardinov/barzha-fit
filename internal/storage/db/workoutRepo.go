package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkoutRepo struct {
	db *pgxpool.Pool
}

func NewWorkoutRepo(db *pgxpool.Pool) *WorkoutRepo { return &WorkoutRepo{db: db} }

func (r *WorkoutRepo) UpsertStatus(ctx context.Context, userID int64, date time.Time, cycleDay int, status string) error {
	_, err := r.db.Exec(ctx, `
		insert into workout_days (user_id, day_date, cycle_day, status)
		values ($1, $2::date, $3, $4)
		on conflict (user_id, day_date)
		do update set cycle_day=excluded.cycle_day, status=excluded.status
	`, userID, date, cycleDay, status)
	return err
}

func (r *WorkoutRepo) GetStatus(ctx context.Context, userID int64, date time.Time) (string, bool, error) {
	var status string
	err := r.db.QueryRow(ctx, `
		select status from workout_days
		where user_id=$1 and day_date=$2::date
	`, userID, date).Scan(&status)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return status, true, nil
}

func (r *WorkoutRepo) GetRecord(ctx context.Context, userID int64, date time.Time) (cycleDay int, status string, ok bool, err error) {
	err = r.db.QueryRow(ctx, `
		select cycle_day, status
		from workout_days
		where user_id=$1 and day_date=$2::date
	`, userID, date).Scan(&cycleDay, &status)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", false, nil
		}
		return 0, "", false, err
	}
	return cycleDay, status, true, nil
}
