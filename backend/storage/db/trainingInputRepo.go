package db

import (
	"context"

	"barzhafit/backend/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TrainingInputRepo struct {
	db *pgxpool.Pool
}

func NewTrainingInputRepo(db *pgxpool.Pool) *TrainingInputRepo { return &TrainingInputRepo{db: db} }

func (r *TrainingInputRepo) Upsert(ctx context.Context, input domain.TrainingInput) (domain.TrainingInput, error) {
	if input.Injuries == nil {
		input.Injuries = []string{}
	}
	var out domain.TrainingInput
	var level string
	var goal string
	err := r.db.QueryRow(ctx, `
		insert into training_inputs (user_id, level, goal, days_per_week, injuries)
		values ($1,$2,$3,$4,$5)
		on conflict (user_id) do update set
			level=excluded.level,
			goal=excluded.goal,
			days_per_week=excluded.days_per_week,
			injuries=excluded.injuries,
			updated_at=now()
		returning id, user_id, level, goal, days_per_week, injuries, created_at, updated_at
	`, input.UserID, input.Level, input.Goal, input.DaysPerWeek, input.Injuries).Scan(
		&out.ID,
		&out.UserID,
		&level,
		&goal,
		&out.DaysPerWeek,
		&out.Injuries,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err == nil {
		out.Level = domain.FitnessLevel(level)
		out.Goal = domain.FitnessGoal(goal)
	}
	return out, err
}

func (r *TrainingInputRepo) GetByUserID(ctx context.Context, userID string) (domain.TrainingInput, bool, error) {
	var out domain.TrainingInput
	var level string
	var goal string
	err := r.db.QueryRow(ctx, `
		select id, user_id, level, goal, days_per_week, injuries, created_at, updated_at
		from training_inputs
		where user_id=$1
	`, userID).Scan(
		&out.ID,
		&out.UserID,
		&level,
		&goal,
		&out.DaysPerWeek,
		&out.Injuries,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.TrainingInput{}, false, nil
		}
		return domain.TrainingInput{}, false, err
	}
	out.Level = domain.FitnessLevel(level)
	out.Goal = domain.FitnessGoal(goal)
	return out, true, nil
}
