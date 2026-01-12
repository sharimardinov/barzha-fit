package db

import (
	"barzhafit/internal/domain"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TargetsRepo struct{ db *pgxpool.Pool }

func NewTargetsRepo(db *pgxpool.Pool) *TargetsRepo { return &TargetsRepo{db: db} }

func (r *TargetsRepo) Upsert(ctx context.Context, t domain.Targets) error {
	_, err := r.db.Exec(ctx, `
		insert into user_targets(chat_id, kcal_target, protein_g, fat_g, carbs_g, source)
		values ($1,$2,$3,$4,$5,$6)
		on conflict (chat_id) do update set
			kcal_target=excluded.kcal_target,
			protein_g=excluded.protein_g,
			fat_g=excluded.fat_g,
			carbs_g=excluded.carbs_g,
			source=excluded.source,
			updated_at=now()
	`, t.ChatID, t.Kcal, t.ProteinG, t.FatG, t.CarbsG, t.Source)
	return err
}

func (r *TargetsRepo) Get(ctx context.Context, chatID int64) (domain.Targets, bool, error) {
	var t domain.Targets
	t.ChatID = chatID

	err := r.db.QueryRow(ctx, `
		select kcal_target, protein_g, fat_g, carbs_g, source
		from user_targets
		where chat_id=$1
	`, chatID).Scan(&t.Kcal, &t.ProteinG, &t.FatG, &t.CarbsG, &t.Source)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Targets{}, false, nil
		}
		return domain.Targets{}, false, err
	}
	return t, true, nil
}

func (r *TargetsRepo) SetManualField(ctx context.Context, chatID int64, field string, value int) error {
	q := ""
	switch field {
	case "kcal":
		q = `update user_targets set kcal_target=$2, source='manual', updated_at=now() where chat_id=$1`
	case "protein":
		q = `update user_targets set protein_g=$2, source='manual', updated_at=now() where chat_id=$1`
	case "fat":
		q = `update user_targets set fat_g=$2, source='manual', updated_at=now() where chat_id=$1`
	case "carbs":
		q = `update user_targets set carbs_g=$2, source='manual', updated_at=now() where chat_id=$1`
	default:
		return fmt.Errorf("unknown field: %s", field)
	}
	_, err := r.db.Exec(ctx, q, chatID, value)
	return err
}
