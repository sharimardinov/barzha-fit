package service

import (
	"barzhafit/internal/domain"
	"barzhafit/internal/util"
	"context"
	"fmt"
	"math"
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
	t, ok, err := s.targets.Get(ctx, chatID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("targets not found")
	}

	switch field {
	case "kcal":
		t.Kcal = value
		t.CarbsG = carbsFromKcal(t.Kcal, t.ProteinG, t.FatG)
		t.Kcal = calcKcal(t.ProteinG, t.FatG, t.CarbsG)
	case "protein":
		t.ProteinG = value
		t.Kcal = calcKcal(t.ProteinG, t.FatG, t.CarbsG)
	case "fat":
		t.FatG = value
		t.Kcal = calcKcal(t.ProteinG, t.FatG, t.CarbsG)
	case "carbs":
		t.CarbsG = value
		t.Kcal = calcKcal(t.ProteinG, t.FatG, t.CarbsG)
	default:
		return fmt.Errorf("unknown field: %s", field)
	}

	t.Source = "manual"
	return s.targets.Upsert(ctx, t)
}

func calcKcal(p, f, c int) int {
	return p*4 + f*9 + c*4
}

func carbsFromKcal(kcal, p, f int) int {
	remaining := kcal - p*4 - f*9
	if remaining <= 0 {
		return 0
	}
	return int(math.Round(float64(remaining) / 4.0))
}
