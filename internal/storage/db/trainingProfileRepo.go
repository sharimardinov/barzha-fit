package db

import (
	"context"
	"database/sql"

	"barzhafit/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TrainingProfileRepo struct{ db *pgxpool.Pool }

func NewTrainingProfileRepo(db *pgxpool.Pool) *TrainingProfileRepo {
	return &TrainingProfileRepo{db: db}
}

func (r *TrainingProfileRepo) Upsert(ctx context.Context, p domain.TrainingProfile) error {
	_, err := r.db.Exec(ctx, `
		insert into training_profiles
			(chat_id, bench_kg, pullups, run_km, injuries, goal, pharma, trainings_per_week, dislikes, cannot_do, wishes)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		on conflict (chat_id) do update set
			bench_kg=excluded.bench_kg,
			pullups=excluded.pullups,
			run_km=excluded.run_km,
			injuries=excluded.injuries,
			goal=excluded.goal,
			pharma=excluded.pharma,
			trainings_per_week=excluded.trainings_per_week,
			dislikes=excluded.dislikes,
			cannot_do=excluded.cannot_do,
			wishes=excluded.wishes,
			updated_at=now()
	`, p.ChatID, nullInt(p.BenchKG), nullInt(p.Pullups), nullFloat(p.RunKM),
		nullStr(p.Injuries), nullStr(p.Goal), nullBoolPtr(p.Pharma), nullInt(p.TrainingsPerWeek),
		nullStr(p.Dislikes), nullStr(p.CannotDo), nullStr(p.Wishes))
	return err
}

func (r *TrainingProfileRepo) Get(ctx context.Context, chatID int64) (domain.TrainingProfile, bool, error) {
	var p domain.TrainingProfile
	p.ChatID = chatID

	var bench sql.NullInt32
	var pullups sql.NullInt32
	var run sql.NullFloat64
	var injuries sql.NullString
	var goal sql.NullString
	var pharma sql.NullBool
	var tpw sql.NullInt32
	var dislikes sql.NullString
	var cannot sql.NullString
	var wishes sql.NullString

	err := r.db.QueryRow(ctx, `
		select bench_kg, pullups, run_km, injuries, goal, pharma, trainings_per_week, dislikes, cannot_do, wishes
		from training_profiles
		where chat_id=$1
	`, chatID).Scan(&bench, &pullups, &run, &injuries, &goal, &pharma, &tpw, &dislikes, &cannot, &wishes)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.TrainingProfile{}, false, nil
		}
		return domain.TrainingProfile{}, false, err
	}

	if bench.Valid {
		p.BenchKG = int(bench.Int32)
	}
	if pullups.Valid {
		p.Pullups = int(pullups.Int32)
	}
	if run.Valid {
		p.RunKM = run.Float64
	}
	if injuries.Valid {
		p.Injuries = injuries.String
	}
	if goal.Valid {
		p.Goal = goal.String
	}
	if pharma.Valid {
		v := pharma.Bool
		p.Pharma = &v
	}
	if tpw.Valid {
		p.TrainingsPerWeek = int(tpw.Int32)
	}
	if dislikes.Valid {
		p.Dislikes = dislikes.String
	}
	if cannot.Valid {
		p.CannotDo = cannot.String
	}
	if wishes.Valid {
		p.Wishes = wishes.String
	}

	return p, true, nil
}

func nullBoolPtr(v *bool) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullFloat(v float64) any {
	if v == 0 {
		return nil
	}
	return v
}
