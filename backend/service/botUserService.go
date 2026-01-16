package service

import "context"

type BotUsersStorage interface {
	Ensure(ctx context.Context, chatID int64) error
	ListEnabled(ctx context.Context) ([]int64, error)
	SetMorning(ctx context.Context, chatID int64, enabled bool) error
	SetHard(ctx context.Context, chatID int64, enabled bool) error
	GetHard(ctx context.Context, chatID int64) (bool, error)
}

type BotUsersService struct {
	repo BotUsersStorage
}

func NewBotUsersService(repo BotUsersStorage) *BotUsersService {
	return &BotUsersService{repo: repo}
}

func (s *BotUsersService) Ensure(ctx context.Context, chatID int64) error {
	return s.repo.Ensure(ctx, chatID)
}

func (s *BotUsersService) ListEnabled(ctx context.Context) ([]int64, error) {
	return s.repo.ListEnabled(ctx)
}

func (s *BotUsersService) SetMorning(ctx context.Context, chatID int64, enabled bool) error {
	return s.repo.SetMorning(ctx, chatID, enabled)
}

func (s *BotUsersService) SetHard(ctx context.Context, chatID int64, enabled bool) error {
	return s.repo.SetHard(ctx, chatID, enabled)
}

func (s *BotUsersService) GetHard(ctx context.Context, chatID int64) (bool, error) {
	return s.repo.GetHard(ctx, chatID)
}
