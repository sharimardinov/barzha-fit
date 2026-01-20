package db

import (
	"context"

	"barzhafit/backend/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkoutSetRepo struct {
	db *pgxpool.Pool
}

func NewWorkoutSetRepo(db *pgxpool.Pool) *WorkoutSetRepo { return &WorkoutSetRepo{db: db} }

func (r *WorkoutSetRepo) Add(ctx context.Context, s *domain.WorkoutSet) error {
	return r.db.QueryRow(ctx, `
		insert into workout_sets(
			session_id, exercise_index, set_index, is_warmup,
			exercise_name, exercise_type,
			target_weight, target_reps, target_duration_sec,
			actual_weight, actual_reps, actual_duration_sec,
			completed_at
		)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		returning id
	`,
		s.SessionID,
		s.ExerciseIndex,
		s.SetIndex,
		s.IsWarmup,
		s.ExerciseName,
		s.ExerciseType,
		s.TargetWeight,
		s.TargetReps,
		s.TargetDurationSec,
		s.ActualWeight,
		s.ActualReps,
		s.ActualDurationSec,
		s.CompletedAt,
	).Scan(&s.ID)
}
