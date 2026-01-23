package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrGoogleAlreadyLinked     = errors.New("google_already_linked")
	ErrGoogleLinkedToOtherUser = errors.New("google_linked_to_other_user")
)

type GoogleAuthRepo struct {
	db *pgxpool.Pool
}

func NewGoogleAuthRepo(db *pgxpool.Pool) *GoogleAuthRepo {
	return &GoogleAuthRepo{db: db}
}

func (r *GoogleAuthRepo) GetChatIDBySub(ctx context.Context, sub string) (int64, bool, error) {
	if sub == "" {
		return 0, false, nil
	}
	var chatID int64
	err := r.db.QueryRow(ctx, `select chat_id from google_auth where google_sub=$1`, sub).Scan(&chatID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return chatID, true, nil
}

func (r *GoogleAuthRepo) GetSubByChatID(ctx context.Context, chatID int64) (string, bool, error) {
	if chatID <= 0 {
		return "", false, nil
	}
	var sub string
	err := r.db.QueryRow(ctx, `select google_sub from google_auth where chat_id=$1`, chatID).Scan(&sub)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return sub, true, nil
}

func (r *GoogleAuthRepo) Link(ctx context.Context, chatID int64, sub, email, name string) error {
	if chatID <= 0 || sub == "" {
		return errors.New("invalid_link")
	}

	var existingChatID int64
	err := r.db.QueryRow(ctx, `select chat_id from google_auth where google_sub=$1`, sub).Scan(&existingChatID)
	if err == nil && existingChatID != chatID {
		return ErrGoogleLinkedToOtherUser
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	var existingSub string
	err = r.db.QueryRow(ctx, `select google_sub from google_auth where chat_id=$1`, chatID).Scan(&existingSub)
	if err == nil && existingSub != sub {
		return ErrGoogleAlreadyLinked
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	_, err = r.db.Exec(ctx, `
		insert into google_auth (google_sub, chat_id, email, name)
		values ($1, $2, $3, $4)
		on conflict (google_sub) do update
			set email=excluded.email,
			    name=excluded.name,
			    updated_at=now()
	`, sub, chatID, email, name)
	return err
}
