package service

import "context"

type BotUsersStorage interface {
	Ensure(ctx context.Context, chatID int64) error
	List(ctx context.Context) ([]int64, error)
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

func (s *BotUsersService) List(ctx context.Context) ([]int64, error) {
	return s.repo.List(ctx)
}
