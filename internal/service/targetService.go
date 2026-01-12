package service

import (
	"barzhafit/internal/domain"
	"barzhafit/internal/util"
	"context"
)

type TargetsStorage interface {
	Upsert(ctx context.Context, t domain.Targets) error
	Get(ctx context.Context, chatID int64) (domain.Targets, bool, error)
	SetManualField(ctx context.Context, chatID int64, field string, value int) error
}

type TargetsService struct {
	targets  TargetsStorage
	profiles ProfileStorage
}

func NewTargetsService(targets TargetsStorage, profiles ProfileStorage) *TargetsService {
	return &TargetsService{targets: targets, profiles: profiles}
}

func (s *TargetsService) Refresh(ctx context.Context, chatID int64) (domain.Targets, error) {
	p, ok, err := s.profiles.Get(ctx, chatID)
	if err != nil {
		return domain.Targets{}, err
	}
	if !ok {
		return domain.Targets{}, nil
	}
	t := util.CalcTargets(p)
	if err := s.targets.Upsert(ctx, t); err != nil {
		return domain.Targets{}, err
	}
	return t, nil
}

func (s *TargetsService) Get(ctx context.Context, chatID int64) (domain.Targets, bool, error) {
	return s.targets.Get(ctx, chatID)
}

func (s *TargetsService) SetManual(ctx context.Context, chatID int64, field string, value int) error {
	return s.targets.SetManualField(ctx, chatID, field, value)
}
