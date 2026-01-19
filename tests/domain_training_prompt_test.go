package tests

import (
	"testing"

	"barzhafit/backend/domain"
)

func TestBuildTrainingPrompt(t *testing.T) {
	tp := domain.TrainingProfile{
		BenchKG:          100,
		Pullups:          10,
		RunKM:            5.5,
		Injuries:         "",
		Goal:             "strength",
		Pharma:           boolPtr(true),
		TrainingsPerWeek: 4,
		Wishes:           "",
	}
	p := domain.Profile{
		Sex:           "m",
		Age:           28,
		HeightCM:      180,
		WeightKG:      85.5,
		BodyFatPct:    12.5,
		TrainingYears: 3,
	}

	out := domain.BuildTrainingPrompt(p, tp)
	if out.Sex != "мужчина" {
		t.Fatalf("expected sex 'мужчина', got %q", out.Sex)
	}
	if out.Pharma != "да" {
		t.Fatalf("expected pharma 'да', got %q", out.Pharma)
	}
	if out.Injuries != "травм нет" {
		t.Fatalf("expected default injuries, got %q", out.Injuries)
	}
	if out.Preferences == "" || out.Normalized.Preferences.Notes == "" {
		t.Fatal("expected default wishes to be populated")
	}
	if out.Normalized.Experience != "intermediate" {
		t.Fatalf("expected experience 'intermediate', got %q", out.Normalized.Experience)
	}
	if out.Normalized.TrainingDaysPerWeek != 4 {
		t.Fatalf("expected training days 4, got %d", out.Normalized.TrainingDaysPerWeek)
	}
}

func boolPtr(v bool) *bool { return &v }
