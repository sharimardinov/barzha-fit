package db

import (
	"context"
	"database/sql"

	"barzhafit/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProfileRepo struct{ db *pgxpool.Pool }

func NewProfileRepo(db *pgxpool.Pool) *ProfileRepo { return &ProfileRepo{db: db} }

func (r *ProfileRepo) Upsert(ctx context.Context, p domain.Profile) error {
	_, err := r.db.Exec(ctx, `
		insert into user_profiles(chat_id, sex, age, height_cm, weight_kg, bodyfat_pct, activity, goal, training_years)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		on conflict (chat_id) do update set
			sex=excluded.sex,
			age=excluded.age,
			height_cm=excluded.height_cm,
			weight_kg=excluded.weight_kg,
			bodyfat_pct=excluded.bodyfat_pct,
			activity=excluded.activity,
			goal=excluded.goal,
			training_years=excluded.training_years,
			updated_at=now()
	`, p.ChatID,
		nullStr(p.Sex),
		nullInt(p.Age),
		nullInt(p.HeightCM),
		p.WeightKG,
		p.BodyFatPct,
		nullStr(p.Activity),
		nullStr(p.Goal),
		nullInt(p.TrainingYears),
	)
	return err
}

func (r *ProfileRepo) Get(ctx context.Context, chatID int64) (domain.Profile, bool, error) {
	var p domain.Profile
	p.ChatID = chatID

	var sex sql.NullString
	var act sql.NullString
	var goal sql.NullString
	var age sql.NullInt32
	var h sql.NullInt32
	var w float64
	var bf float64
	var years sql.NullInt32

	err := r.db.QueryRow(ctx, `
		select sex, age, height_cm, weight_kg, bodyfat_pct, activity, goal, training_years
		from user_profiles
		where chat_id=$1
	`, chatID).Scan(&sex, &age, &h, &w, &bf, &act, &goal, &years)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Profile{}, false, nil
		}
		return domain.Profile{}, false, err
	}

	if sex.Valid {
		p.Sex = sex.String
	}
	if age.Valid {
		p.Age = int(age.Int32)
	}
	if h.Valid {
		p.HeightCM = int(h.Int32)
	}
	p.WeightKG = w
	p.BodyFatPct = bf
	if act.Valid {
		p.Activity = act.String
	}
	if p.Activity == "" {
		p.Activity = "mid"
	}
	if goal.Valid {
		p.Goal = goal.String
	}
	if p.Goal == "" {
		p.Goal = "balance"
	}
	if years.Valid {
		p.TrainingYears = int(years.Int32)
	}

	return p, true, nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}
