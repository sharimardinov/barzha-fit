package service

import "context"

type GoogleAuthStorage interface {
	Link(ctx context.Context, chatID int64, sub, email, name string) error
	GetChatIDBySub(ctx context.Context, sub string) (int64, bool, error)
	EnsureBySub(ctx context.Context, sub, email, name string) (int64, bool, error)
}

type GoogleAuthService struct {
	repo GoogleAuthStorage
}

func NewGoogleAuthService(repo GoogleAuthStorage) *GoogleAuthService {
	return &GoogleAuthService{repo: repo}
}

func (s *GoogleAuthService) Link(ctx context.Context, chatID int64, sub, email, name string) error {
	return s.repo.Link(ctx, chatID, sub, email, name)
}

func (s *GoogleAuthService) ResolveChatID(ctx context.Context, sub string) (int64, bool, error) {
	return s.repo.GetChatIDBySub(ctx, sub)
}

func (s *GoogleAuthService) EnsureChatIDBySub(ctx context.Context, sub, email, name string) (int64, bool, error) {
	return s.repo.EnsureBySub(ctx, sub, email, name)
}
