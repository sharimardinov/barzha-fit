package db

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (r *GoogleAuthRepo) EnsureBySub(ctx context.Context, sub, email, name string) (int64, bool, error) {
	if sub == "" {
		return 0, false, errors.New("invalid_sub")
	}
	chatID, ok, err := r.GetChatIDBySub(ctx, sub)
	if err != nil {
		return 0, false, err
	}
	if ok {
		return chatID, false, nil
	}

	for i := 0; i < 8; i += 1 {
		chatID, err = generateNegativeID()
		if err != nil {
			return 0, false, err
		}
		_, err = r.db.Exec(ctx, `
			insert into google_auth (google_sub, chat_id, email, name)
			values ($1, $2, $3, $4)
		`, sub, chatID, email, name)
		if err == nil {
			return chatID, true, nil
		}
		if isUniqueViolation(err) {
			existing, ok, err := r.GetChatIDBySub(ctx, sub)
			if err != nil {
				return 0, false, err
			}
			if ok {
				return existing, false, nil
			}
			continue
		}
		return 0, false, err
	}
	return 0, false, errors.New("google_user_create_failed")
}

func generateNegativeID() (int64, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, err
	}
	val := int64(binary.BigEndian.Uint64(buf[:]))
	if val == 0 {
		val = 1
	}
	if val > 0 {
		val = -val
	}
	return val, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505"
}
