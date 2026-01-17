package tests

import (
	"context"
	"testing"

	"barzhafit/backend/domain"
	"barzhafit/backend/service"
)

func TestNormalizeFitnessLevel(t *testing.T) {
	tests := []struct {
		in   string
		want domain.FitnessLevel
		ok   bool
	}{
		{"core", domain.FitnessBeginner, true},
		{"beginner", domain.FitnessBeginner, true},
		{"FLOW", domain.FitnessIntermediate, true},
		{"peak", domain.FitnessAdvanced, true},
		{"unknown", "", false},
	}
	for _, tt := range tests {
		got, ok := domain.NormalizeFitnessLevel(tt.in)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Fatalf("NormalizeFitnessLevel(%q) = (%q,%v), want (%q,%v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestNormalizeFitnessGoal(t *testing.T) {
	tests := []struct {
		in   string
		want domain.FitnessGoal
		ok   bool
	}{
		{"cut", domain.GoalFatLoss, true},
		{"fat_loss", domain.GoalFatLoss, true},
		{"bulk", domain.GoalHypertrophy, true},
		{"hypertrophy", domain.GoalHypertrophy, true},
		{"balance", domain.GoalStrength, true},
		{"strength", domain.GoalStrength, true},
		{"unknown", "", false},
	}
	for _, tt := range tests {
		got, ok := domain.NormalizeFitnessGoal(tt.in)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Fatalf("NormalizeFitnessGoal(%q) = (%q,%v), want (%q,%v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestTrainingInputServiceSaveFromSelection(t *testing.T) {
	userRepo := &fakeUserRepo{userID: "user-123"}
	inputRepo := &fakeTrainingInputRepo{}
	svc := service.NewTrainingInputService(inputRepo, userRepo)

	got, err := svc.SaveFromSelection(context.Background(), 42, "core", "cut", 4, []string{" Shoulder ", "", "Lower_Back"})
	if err != nil {
		t.Fatalf("SaveFromSelection error: %v", err)
	}
	if userRepo.chatID != 42 {
		t.Fatalf("expected chat_id 42, got %d", userRepo.chatID)
	}
	if inputRepo.saved.UserID != "user-123" {
		t.Fatalf("expected user_id user-123, got %q", inputRepo.saved.UserID)
	}
	if inputRepo.saved.Level != domain.FitnessBeginner {
		t.Fatalf("expected level beginner, got %q", inputRepo.saved.Level)
	}
	if inputRepo.saved.Goal != domain.GoalFatLoss {
		t.Fatalf("expected goal fat_loss, got %q", inputRepo.saved.Goal)
	}
	if inputRepo.saved.DaysPerWeek != 4 {
		t.Fatalf("expected days 4, got %d", inputRepo.saved.DaysPerWeek)
	}
	if len(inputRepo.saved.Injuries) != 2 || inputRepo.saved.Injuries[0] != "shoulder" || inputRepo.saved.Injuries[1] != "lower_back" {
		t.Fatalf("unexpected injuries: %v", inputRepo.saved.Injuries)
	}
	if got.UserID != "user-123" {
		t.Fatalf("expected returned user_id user-123, got %q", got.UserID)
	}
}

func TestTrainingInputServiceValidation(t *testing.T) {
	userRepo := &fakeUserRepo{userID: "user-123"}
	inputRepo := &fakeTrainingInputRepo{}
	svc := service.NewTrainingInputService(inputRepo, userRepo)

	if _, err := svc.SaveFromSelection(context.Background(), 0, "core", "cut", 4, nil); err == nil {
		t.Fatal("expected error for invalid chat_id")
	}
	if _, err := svc.SaveFromSelection(context.Background(), 42, "unknown", "cut", 4, nil); err == nil {
		t.Fatal("expected error for invalid fitness_level")
	}
	if _, err := svc.SaveFromSelection(context.Background(), 42, "core", "unknown", 4, nil); err == nil {
		t.Fatal("expected error for invalid goal")
	}
	if _, err := svc.SaveFromSelection(context.Background(), 42, "core", "cut", 1, nil); err == nil {
		t.Fatal("expected error for invalid days_per_week")
	}
}

type fakeUserRepo struct {
	userID string
	chatID int64
	err    error
}

func (f *fakeUserRepo) EnsureByTelegramChatID(ctx context.Context, chatID int64) (string, error) {
	f.chatID = chatID
	return f.userID, f.err
}

type fakeTrainingInputRepo struct {
	saved domain.TrainingInput
	err   error
}

func (f *fakeTrainingInputRepo) Upsert(ctx context.Context, input domain.TrainingInput) (domain.TrainingInput, error) {
	f.saved = input
	return input, f.err
}
