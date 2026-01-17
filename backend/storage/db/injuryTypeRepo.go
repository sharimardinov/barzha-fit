package db

import (
	"context"

	"barzhafit/backend/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type InjuryTypeRepo struct {
	db *pgxpool.Pool
}

func NewInjuryTypeRepo(db *pgxpool.Pool) *InjuryTypeRepo { return &InjuryTypeRepo{db: db} }

func (r *InjuryTypeRepo) List(ctx context.Context) ([]domain.InjuryType, error) {
	rows, err := r.db.Query(ctx, `
		select code, label
		from injury_types
		order by label
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.InjuryType, 0)
	for rows.Next() {
		var item domain.InjuryType
		if err := rows.Scan(&item.Code, &item.Label); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
