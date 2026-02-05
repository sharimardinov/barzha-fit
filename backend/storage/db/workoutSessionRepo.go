package db

import (
	"context"
	"database/sql"

	"barzhafit/backend/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkoutSessionRepo struct {
	db *pgxpool.Pool
}

func NewWorkoutSessionRepo(db *pgxpool.Pool) *WorkoutSessionRepo { return &WorkoutSessionRepo{db: db} }

func (r *WorkoutSessionRepo) GetActive(ctx context.Context, chatID int64) (domain.WorkoutSession, bool, error) {
	var s domain.WorkoutSession
	var planID sql.NullInt64
	var timerKind sql.NullString
	var timerStarted sql.NullTime
	var warmupEnded sql.NullTime
	var pausedAt sql.NullTime
	var timerDuration sql.NullInt64

	err := r.db.QueryRow(ctx, `
		select id, chat_id, plan_id, plan_snapshot, status, phase,
		       exercise_index, set_index, timer_kind, timer_started_at,
		       timer_duration_sec, warmup_ended_at, paused_at,
		       paused_total_sec, started_at, updated_at
		from workout_sessions
		where chat_id=$1 and status in ($2, $3)
		order by id desc
		limit 1
	`, chatID, domain.WorkoutSessionStatusInProgress, domain.WorkoutSessionStatusPaused).Scan(
		&s.ID,
		&s.ChatID,
		&planID,
		&s.PlanSnapshot,
		&s.Status,
		&s.Phase,
		&s.ExerciseIndex,
		&s.SetIndex,
		&timerKind,
		&timerStarted,
		&timerDuration,
		&warmupEnded,
		&pausedAt,
		&s.PausedTotalSec,
		&s.StartedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.WorkoutSession{}, false, nil
		}
		return domain.WorkoutSession{}, false, err
	}
	if planID.Valid {
		v := planID.Int64
		s.PlanID = &v
	}
	if timerKind.Valid {
		s.TimerKind = domain.WorkoutTimerKind(timerKind.String)
	}
	if timerStarted.Valid {
		t := timerStarted.Time
		s.TimerStartedAt = &t
	}
	if timerDuration.Valid {
		s.TimerDurationSec = int(timerDuration.Int64)
	}
	if warmupEnded.Valid {
		t := warmupEnded.Time
		s.WarmupEndedAt = &t
	}
	if pausedAt.Valid {
		t := pausedAt.Time
		s.PausedAt = &t
	}
	return s, true, nil
}

func (r *WorkoutSessionRepo) Create(ctx context.Context, s *domain.WorkoutSession) error {
	var id int64
	var startedAt sql.NullTime
	var updatedAt sql.NullTime
	var planID sql.NullInt64
	if s.PlanID != nil {
		planID.Valid = true
		planID.Int64 = *s.PlanID
	}
	var timerKind sql.NullString
	if s.TimerKind != "" {
		timerKind.Valid = true
		timerKind.String = string(s.TimerKind)
	}
	var timerStarted sql.NullTime
	if s.TimerStartedAt != nil {
		timerStarted.Valid = true
		timerStarted.Time = *s.TimerStartedAt
	}
	var timerDuration sql.NullInt64
	if s.TimerDurationSec > 0 {
		timerDuration.Valid = true
		timerDuration.Int64 = int64(s.TimerDurationSec)
	}
	var warmupEnded sql.NullTime
	if s.WarmupEndedAt != nil {
		warmupEnded.Valid = true
		warmupEnded.Time = *s.WarmupEndedAt
	}
	var pausedAt sql.NullTime
	if s.PausedAt != nil {
		pausedAt.Valid = true
		pausedAt.Time = *s.PausedAt
	}

	err := r.db.QueryRow(ctx, `
		insert into workout_sessions(
			chat_id, plan_id, plan_snapshot, status, phase,
			exercise_index, set_index, timer_kind, timer_started_at,
			timer_duration_sec, warmup_ended_at, paused_at,
			paused_total_sec
		)
		values ($1,$2,$3::jsonb,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		returning id, started_at, updated_at
	`,
		s.ChatID,
		planID,
		string(s.PlanSnapshot),
		s.Status,
		s.Phase,
		s.ExerciseIndex,
		s.SetIndex,
		timerKind,
		timerStarted,
		timerDuration,
		warmupEnded,
		pausedAt,
		s.PausedTotalSec,
	).Scan(&id, &startedAt, &updatedAt)
	if err != nil {
		return err
	}
	s.ID = id
	if startedAt.Valid {
		s.StartedAt = startedAt.Time
	}
	if updatedAt.Valid {
		s.UpdatedAt = updatedAt.Time
	}
	return nil
}

func (r *WorkoutSessionRepo) Update(ctx context.Context, s *domain.WorkoutSession) error {
	var planID sql.NullInt64
	if s.PlanID != nil {
		planID.Valid = true
		planID.Int64 = *s.PlanID
	}
	var timerKind sql.NullString
	if s.TimerKind != "" {
		timerKind.Valid = true
		timerKind.String = string(s.TimerKind)
	}
	var timerStarted sql.NullTime
	if s.TimerStartedAt != nil {
		timerStarted.Valid = true
		timerStarted.Time = *s.TimerStartedAt
	}
	var timerDuration sql.NullInt64
	if s.TimerDurationSec > 0 {
		timerDuration.Valid = true
		timerDuration.Int64 = int64(s.TimerDurationSec)
	}
	var warmupEnded sql.NullTime
	if s.WarmupEndedAt != nil {
		warmupEnded.Valid = true
		warmupEnded.Time = *s.WarmupEndedAt
	}
	var pausedAt sql.NullTime
	if s.PausedAt != nil {
		pausedAt.Valid = true
		pausedAt.Time = *s.PausedAt
	}

	_, err := r.db.Exec(ctx, `
		update workout_sessions
		set chat_id=$2,
			plan_id=$3,
			plan_snapshot=$4::jsonb,
			status=$5,
			phase=$6,
			exercise_index=$7,
			set_index=$8,
			timer_kind=$9,
			timer_started_at=$10,
			timer_duration_sec=$11,
			warmup_ended_at=$12,
			paused_at=$13,
			paused_total_sec=$14,
			updated_at=now()
		where id=$1
	`,
		s.ID,
		s.ChatID,
		planID,
		string(s.PlanSnapshot),
		s.Status,
		s.Phase,
		s.ExerciseIndex,
		s.SetIndex,
		timerKind,
		timerStarted,
		timerDuration,
		warmupEnded,
		pausedAt,
		s.PausedTotalSec,
	)
	return err
}
