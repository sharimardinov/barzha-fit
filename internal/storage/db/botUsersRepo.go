package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BotUsersRepo struct{ db *pgxpool.Pool }

func NewBotUsersRepo(db *pgxpool.Pool) *BotUsersRepo { return &BotUsersRepo{db: db} }

func (r *BotUsersRepo) Ensure(ctx context.Context, chatID int64) error {
	_, err := r.db.Exec(ctx, `
		insert into bot_users(chat_id, morning_enabled, hard_enabled)
		values ($1, true, false)
		on conflict (chat_id) do nothing
	`, chatID)
	return err
}

func (r *BotUsersRepo) ListEnabled(ctx context.Context) ([]int64, error) {
	rows, err := r.db.Query(ctx, `
		select chat_id
		from bot_users
		where morning_enabled = true
		order by chat_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		res = append(res, id)
	}
	return res, rows.Err()
}

func (r *BotUsersRepo) SetMorning(ctx context.Context, chatID int64, enabled bool) error {
	_, err := r.db.Exec(ctx, `
		update bot_users
		set morning_enabled = $2
		where chat_id = $1
	`, chatID, enabled)
	return err
}

func (r *BotUsersRepo) SetHard(ctx context.Context, chatID int64, enabled bool) error {
	_, err := r.db.Exec(ctx, `
		update bot_users
		set hard_enabled = $2
		where chat_id = $1
	`, chatID, enabled)
	return err
}

func (r *BotUsersRepo) GetHard(ctx context.Context, chatID int64) (bool, error) {
	var v bool
	err := r.db.QueryRow(ctx, `
		select hard_enabled
		from bot_users
		where chat_id = $1
	`, chatID).Scan(&v)
	if err != nil {
		return false, err
	}
	return v, nil
}
