package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AppUserRepo struct {
	db *pgxpool.Pool
}

func NewAppUserRepo(db *pgxpool.Pool) *AppUserRepo { return &AppUserRepo{db: db} }

func (r *AppUserRepo) EnsureByTelegramChatID(ctx context.Context, chatID int64) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		insert into users (telegram_chat_id)
		values ($1)
		on conflict (telegram_chat_id) do update set updated_at=now()
		returning id
	`, chatID).Scan(&id)
	return id, err
}
