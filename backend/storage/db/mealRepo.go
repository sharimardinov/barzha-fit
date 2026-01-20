package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Meal struct {
	ID       int64
	ChatID   int64
	EatenAt  time.Time
	Text     string
	Kcal     int
	ProteinG int
	FatG     int
	CarbsG   int
}

type MealRepo struct{ db *pgxpool.Pool }

func NewMealRepo(db *pgxpool.Pool) *MealRepo { return &MealRepo{db: db} }

func (r *MealRepo) Add(ctx context.Context, m *Meal, aiRaw any) error {
	var rawBytes []byte
	if aiRaw != nil {
		b, err := json.Marshal(aiRaw)
		if err != nil {
			return err
		}
		rawBytes = b
	}

	err := r.db.QueryRow(ctx, `
		insert into meals(chat_id, eaten_at, text, kcal, protein_g, fat_g, carbs_g, ai_raw)
		values ($1,$2,$3,$4,$5,$6,$7,$8::jsonb)
		returning id
	`, m.ChatID, m.EatenAt, m.Text, m.Kcal, m.ProteinG, m.FatG, m.CarbsG, rawJSON(rawBytes)).Scan(&m.ID)

	return err
}

func rawJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b) // pgx нормально кастит string в jsonb через ::jsonb
}

func (r *MealRepo) DeleteLast(ctx context.Context, chatID int64) (bool, error) {
	ct, err := r.db.Exec(ctx, `
		delete from meals
		where id = (
			select id from meals
			where chat_id=$1
			order by eaten_at desc, id desc
			limit 1
		)
	`, chatID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

func (r *MealRepo) DeleteByID(ctx context.Context, chatID int64, id int64) (bool, error) {
	ct, err := r.db.Exec(ctx, `
		delete from meals
		where chat_id=$1 and id=$2
	`, chatID, id)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

func (r *MealRepo) ListByDay(ctx context.Context, chatID int64, from, to time.Time) ([]Meal, error) {
	rows, err := r.db.Query(ctx, `
		select id, chat_id, eaten_at, text, kcal, protein_g, fat_g, carbs_g
		from meals
		where chat_id=$1 and eaten_at >= $2 and eaten_at < $3
		order by eaten_at asc, id asc
	`, chatID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Meal
	for rows.Next() {
		var m Meal
		if err := rows.Scan(&m.ID, &m.ChatID, &m.EatenAt, &m.Text, &m.Kcal, &m.ProteinG, &m.FatG, &m.CarbsG); err != nil {
			return nil, err
		}
		res = append(res, m)
	}
	return res, rows.Err()
}

func (r *MealRepo) ListRecent(ctx context.Context, chatID int64, limit int) ([]Meal, error) {
	rows, err := r.db.Query(ctx, `
		select id, chat_id, eaten_at, text, kcal, protein_g, fat_g, carbs_g
		from meals
		where chat_id=$1
		order by eaten_at desc, id desc
		limit $2
	`, chatID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Meal
	for rows.Next() {
		var m Meal
		if err := rows.Scan(&m.ID, &m.ChatID, &m.EatenAt, &m.Text, &m.Kcal, &m.ProteinG, &m.FatG, &m.CarbsG); err != nil {
			return nil, err
		}
		res = append(res, m)
	}
	return res, rows.Err()
}

func (r *MealRepo) SumByDay(ctx context.Context, chatID int64, from, to time.Time) (kcal, p, f, c int, err error) {
	err = r.db.QueryRow(ctx, `
		select
			coalesce(sum(protein_g*4 + fat_g*9 + carbs_g*4),0),
			coalesce(sum(protein_g),0),
			coalesce(sum(fat_g),0),
			coalesce(sum(carbs_g),0)
		from meals
		where chat_id=$1 and eaten_at >= $2 and eaten_at < $3
	`, chatID, from, to).Scan(&kcal, &p, &f, &c)
	return
}

func (r *MealRepo) SumAllTime(ctx context.Context, chatID int64) (kcal, p, f, c int, err error) {
	err = r.db.QueryRow(ctx, `
		select
			coalesce(sum(protein_g*4 + fat_g*9 + carbs_g*4),0),
			coalesce(sum(protein_g),0),
			coalesce(sum(fat_g),0),
			coalesce(sum(carbs_g),0)
		from meals
		where chat_id=$1
	`, chatID).Scan(&kcal, &p, &f, &c)
	return
}

type DayNutrition struct {
	Kcal int
	P    int
	F    int
	C    int
}

func (r *MealRepo) SumByRangeDaily(ctx context.Context, chatID int64, from, to time.Time, tz string) (map[string]DayNutrition, error) {
	rows, err := r.db.Query(ctx, `
		select
			(eaten_at at time zone $4)::date as day_date,
			coalesce(sum(protein_g*4 + fat_g*9 + carbs_g*4),0),
			coalesce(sum(protein_g),0),
			coalesce(sum(fat_g),0),
			coalesce(sum(carbs_g),0)
		from meals
		where chat_id=$1 and eaten_at >= $2 and eaten_at < $3
		group by day_date
		order by day_date asc
	`, chatID, from, to, tz)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]DayNutrition)
	for rows.Next() {
		var day time.Time
		var dn DayNutrition
		if err := rows.Scan(&day, &dn.Kcal, &dn.P, &dn.F, &dn.C); err != nil {
			return nil, err
		}
		res[day.Format("2006-01-02")] = dn
	}
	return res, rows.Err()
}
