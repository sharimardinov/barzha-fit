package service

import (
	"context"

	"barzhafit/backend/domain"
)

type InjuryTypeStorage interface {
	List(ctx context.Context) ([]domain.InjuryType, error)
}

type InjuryTypeService struct {
	repo InjuryTypeStorage
}

func NewInjuryTypeService(repo InjuryTypeStorage) *InjuryTypeService {
	return &InjuryTypeService{repo: repo}
}

func (s *InjuryTypeService) List(ctx context.Context) ([]domain.InjuryType, error) {
	return s.repo.List(ctx)
}
