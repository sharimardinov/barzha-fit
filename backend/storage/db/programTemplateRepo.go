package db

import (
	"context"
	"encoding/json"

	"barzhafit/backend/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProgramTemplateRepo struct {
	db *pgxpool.Pool
}

func NewProgramTemplateRepo(db *pgxpool.Pool) *ProgramTemplateRepo {
	return &ProgramTemplateRepo{db: db}
}

func (r *ProgramTemplateRepo) GetByName(ctx context.Context, name string) (domain.ProgramTemplate, bool, error) {
	var out domain.ProgramTemplate
	var structure []byte
	err := r.db.QueryRow(ctx, `
		select id, name, days, structure, created_at, updated_at
		from program_templates
		where name=$1
	`, name).Scan(
		&out.ID,
		&out.Name,
		&out.Days,
		&structure,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ProgramTemplate{}, false, nil
		}
		return domain.ProgramTemplate{}, false, err
	}
	if len(structure) > 0 {
		_ = json.Unmarshal(structure, &out.Structure)
	}
	return out, true, nil
}
