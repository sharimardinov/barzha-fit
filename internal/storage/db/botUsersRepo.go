package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BotUsersRepo struct{ db *pgxpool.Pool }

func NewBotUsersRepo(db *pgxpool.Pool) *BotUsersRepo { return &BotUsersRepo{db: db} }

func (r *BotUsersRepo) Ensure(ctx context.Context, chatID int64) error {
	_, err := r.db.Exec(ctx, `
		insert into bot_users(chat_id) values ($1)
		on conflict (chat_id) do nothing
	`, chatID)
	return err
}

func (r *BotUsersRepo) List(ctx context.Context) ([]int64, error) {
	rows, err := r.db.Query(ctx, `select chat_id from bot_users order by chat_id`)
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
