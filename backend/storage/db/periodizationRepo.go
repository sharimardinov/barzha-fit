package db

import (
	"context"

	"barzhafit/backend/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PeriodizationRepo struct {
	db *pgxpool.Pool
}

func NewPeriodizationRepo(db *pgxpool.Pool) *PeriodizationRepo { return &PeriodizationRepo{db: db} }

func (r *PeriodizationRepo) GetByWeek(ctx context.Context, week int) (domain.Periodization, bool, error) {
	var out domain.Periodization
	err := r.db.QueryRow(ctx, `
		select week, intensity, percent_1rm, reps, rest
		from periodization
		where week=$1
	`, week).Scan(&out.Week, &out.Intensity, &out.Percent1RM, &out.Reps, &out.Rest)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Periodization{}, false, nil
		}
		return domain.Periodization{}, false, err
	}
	return out, true, nil
}
