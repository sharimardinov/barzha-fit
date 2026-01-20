package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StepsRepo struct {
	db *pgxpool.Pool
}

func NewStepsRepo(db *pgxpool.Pool) *StepsRepo { return &StepsRepo{db: db} }

func (r *StepsRepo) Upsert(ctx context.Context, userID int64, dayDate string, steps int) error {
	_, err := r.db.Exec(ctx, `
		insert into steps_days (user_id, day_date, steps)
		values ($1, $2::date, $3)
		on conflict (user_id, day_date)
		do update set steps=excluded.steps
	`, userID, dayDate, steps)
	return err
}

func (r *StepsRepo) Get(ctx context.Context, userID int64, dayDate string) (int, bool, error) {
	var steps int
	err := r.db.QueryRow(ctx, `
		select steps
		from steps_days
		where user_id=$1 and day_date=$2::date
	`, userID, dayDate).Scan(&steps)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return steps, true, nil
}

func (r *StepsRepo) ListByRange(ctx context.Context, userID int64, fromDate, toDate string) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `
		select day_date, steps
		from steps_days
		where user_id=$1 and day_date >= $2::date and day_date <= $3::date
		order by day_date asc
	`, userID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]int)
	for rows.Next() {
		var day time.Time
		var steps int
		if err := rows.Scan(&day, &steps); err != nil {
			return nil, err
		}
		res[day.Format("2006-01-02")] = steps
	}
	return res, rows.Err()
}

func (r *StepsRepo) SumAllTime(ctx context.Context, userID int64) (int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		select coalesce(sum(steps), 0)
		from steps_days
		where user_id=$1
	`, userID).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}
