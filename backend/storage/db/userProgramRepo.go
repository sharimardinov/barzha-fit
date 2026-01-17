package db

import (
	"context"

	"barzhafit/backend/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserProgramRepo struct {
	db *pgxpool.Pool
}

func NewUserProgramRepo(db *pgxpool.Pool) *UserProgramRepo { return &UserProgramRepo{db: db} }

func (r *UserProgramRepo) GetLatestByUserID(ctx context.Context, userID string) (domain.UserProgram, bool, error) {
	var out domain.UserProgram
	err := r.db.QueryRow(ctx, `
		select id, user_id, template_id, start_date, current_week, days_generated, created_at, updated_at
		from user_programs
		where user_id=$1
		order by created_at desc
		limit 1
	`, userID).Scan(
		&out.ID,
		&out.UserID,
		&out.TemplateID,
		&out.StartDate,
		&out.CurrentWeek,
		&out.DaysGenerated,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.UserProgram{}, false, nil
		}
		return domain.UserProgram{}, false, err
	}
	return out, true, nil
}

func (r *UserProgramRepo) Insert(ctx context.Context, program domain.UserProgram) (domain.UserProgram, error) {
	var out domain.UserProgram
	err := r.db.QueryRow(ctx, `
		insert into user_programs (user_id, template_id, start_date, current_week, days_generated)
		values ($1,$2,$3,$4,$5)
		returning id, user_id, template_id, start_date, current_week, days_generated, created_at, updated_at
	`, program.UserID, program.TemplateID, program.StartDate, program.CurrentWeek, program.DaysGenerated).Scan(
		&out.ID,
		&out.UserID,
		&out.TemplateID,
		&out.StartDate,
		&out.CurrentWeek,
		&out.DaysGenerated,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return domain.UserProgram{}, err
	}
	return out, nil
}

func (r *UserProgramRepo) Update(ctx context.Context, program domain.UserProgram) (domain.UserProgram, error) {
	var out domain.UserProgram
	err := r.db.QueryRow(ctx, `
		update user_programs
		set current_week=$2,
		    days_generated=$3,
		    updated_at=now()
		where id=$1
		returning id, user_id, template_id, start_date, current_week, days_generated, created_at, updated_at
	`, program.ID, program.CurrentWeek, program.DaysGenerated).Scan(
		&out.ID,
		&out.UserID,
		&out.TemplateID,
		&out.StartDate,
		&out.CurrentWeek,
		&out.DaysGenerated,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return domain.UserProgram{}, err
	}
	return out, nil
}
