package db

import (
	"context"
	"time"

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

func (r *WorkoutSetRepo) ListBySession(ctx context.Context, sessionID int64) ([]domain.WorkoutSet, error) {
	rows, err := r.db.Query(ctx, `
		select id, session_id, exercise_index, set_index, is_warmup,
		       exercise_name, exercise_type,
		       target_weight, target_reps, target_duration_sec,
		       actual_weight, actual_reps, actual_duration_sec,
		       completed_at
		from workout_sets
		where session_id=$1
		order by exercise_index asc, set_index asc, id asc
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.WorkoutSet, 0)
	for rows.Next() {
		var item domain.WorkoutSet
		if err := rows.Scan(
			&item.ID,
			&item.SessionID,
			&item.ExerciseIndex,
			&item.SetIndex,
			&item.IsWarmup,
			&item.ExerciseName,
			&item.ExerciseType,
			&item.TargetWeight,
			&item.TargetReps,
			&item.TargetDurationSec,
			&item.ActualWeight,
			&item.ActualReps,
			&item.ActualDurationSec,
			&item.CompletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
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

func (r *WorkoutSetRepo) StrengthExerciseNames(ctx context.Context, chatID int64, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		select ws.exercise_name
		from workout_sets ws
		join workout_sessions s on s.id = ws.session_id
		where s.chat_id=$1
		  and ws.exercise_type=$2
		  and ws.is_warmup=false
		group by ws.exercise_name
		order by max(ws.completed_at) desc
		limit $3
	`, chatID, domain.WorkoutExerciseStrength, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func (r *WorkoutSetRepo) StrengthEntriesByExerciseSince(ctx context.Context, chatID int64, exerciseName string, since time.Time, limit int) ([]domain.StrengthExerciseEntry, error) {
	rows, err := r.db.Query(ctx, `
		select actual_weight, actual_reps, completed_at
		from workout_sets ws
		join workout_sessions s on s.id = ws.session_id
		where s.chat_id=$1
		  and ws.exercise_type=$2
		  and ws.is_warmup=false
		  and ws.exercise_name=$3
		  and ws.completed_at >= $4
		order by ws.completed_at asc
		limit $5
	`, chatID, domain.WorkoutExerciseStrength, exerciseName, since, limit)
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
