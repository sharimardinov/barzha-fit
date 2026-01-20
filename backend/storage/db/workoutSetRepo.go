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

func (r *WorkoutSetRepo) StrengthTotals(ctx context.Context, chatID int64) (domain.StrengthTotals, error) {
	var out domain.StrengthTotals
	err := r.db.QueryRow(ctx, `
		select
			count(*)::int,
			coalesce(sum(actual_reps), 0)::int,
			coalesce(sum(actual_weight * actual_reps), 0)::float8,
			coalesce(avg(nullif(actual_weight, 0)), 0)::float8,
			coalesce(avg(nullif(actual_reps, 0)), 0)::float8,
			coalesce(max(actual_weight), 0)::float8
		from workout_sets ws
		join workout_sessions s on s.id = ws.session_id
		where s.chat_id=$1
		  and ws.exercise_type=$2
		  and ws.is_warmup=false
	`, chatID, domain.WorkoutExerciseStrength).Scan(
		&out.Sets,
		&out.Reps,
		&out.Tonnage,
		&out.AvgWeight,
		&out.AvgReps,
		&out.MaxWeight,
	)
	if err != nil {
		return domain.StrengthTotals{}, err
	}
	return out, nil
}

func (r *WorkoutSetRepo) StrengthTopByTonnage(ctx context.Context, chatID int64, limit int) ([]domain.StrengthExerciseStats, error) {
	rows, err := r.db.Query(ctx, `
		select
			exercise_name,
			count(*)::int,
			coalesce(sum(actual_reps), 0)::int,
			coalesce(sum(actual_weight * actual_reps), 0)::float8,
			coalesce(max(actual_weight), 0)::float8
		from workout_sets ws
		join workout_sessions s on s.id = ws.session_id
		where s.chat_id=$1
		  and ws.exercise_type=$2
		  and ws.is_warmup=false
		group by exercise_name
		order by coalesce(sum(actual_weight * actual_reps), 0) desc
		limit $3
	`, chatID, domain.WorkoutExerciseStrength, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.StrengthExerciseStats, 0)
	for rows.Next() {
		var item domain.StrengthExerciseStats
		if err := rows.Scan(&item.Name, &item.Sets, &item.Reps, &item.Tonnage, &item.MaxWeight); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *WorkoutSetRepo) StrengthTopByReps(ctx context.Context, chatID int64, limit int) ([]domain.StrengthExerciseStats, error) {
	rows, err := r.db.Query(ctx, `
		select
			exercise_name,
			count(*)::int,
			coalesce(sum(actual_reps), 0)::int,
			coalesce(sum(actual_weight * actual_reps), 0)::float8,
			coalesce(max(actual_weight), 0)::float8
		from workout_sets ws
		join workout_sessions s on s.id = ws.session_id
		where s.chat_id=$1
		  and ws.exercise_type=$2
		  and ws.is_warmup=false
		group by exercise_name
		order by coalesce(sum(actual_reps), 0) desc
		limit $3
	`, chatID, domain.WorkoutExerciseStrength, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.StrengthExerciseStats, 0)
	for rows.Next() {
		var item domain.StrengthExerciseStats
		if err := rows.Scan(&item.Name, &item.Sets, &item.Reps, &item.Tonnage, &item.MaxWeight); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *WorkoutSetRepo) StrengthRecentEntries(ctx context.Context, chatID int64, exerciseName string, limit int) ([]domain.StrengthExerciseEntry, error) {
	rows, err := r.db.Query(ctx, `
		select actual_weight, actual_reps, completed_at
		from workout_sets ws
		join workout_sessions s on s.id = ws.session_id
		where s.chat_id=$1
		  and ws.exercise_type=$2
		  and ws.is_warmup=false
		  and ws.exercise_name=$3
		order by completed_at desc
		limit $4
	`, chatID, domain.WorkoutExerciseStrength, exerciseName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.StrengthExerciseEntry, 0)
	for rows.Next() {
		var item domain.StrengthExerciseEntry
		if err := rows.Scan(&item.Weight, &item.Reps, &item.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
