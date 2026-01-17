package tests

import (
	"context"
	"testing"
	"time"

	"barzhafit/backend/domain"
	"barzhafit/backend/service"
)

func TestSelectTemplateName(t *testing.T) {
	tests := []struct {
		days  int
		level domain.FitnessLevel
		want  string
		ok    bool
	}{
		{2, domain.FitnessBeginner, "full_body", true},
		{3, domain.FitnessBeginner, "full_body", true},
		{3, domain.FitnessIntermediate, "push_pull_legs", true},
		{4, domain.FitnessAdvanced, "upper_lower_x2", true},
		{5, domain.FitnessBeginner, "upper_lower_arm_day", true},
		{6, domain.FitnessIntermediate, "ppl_x2", true},
		{1, domain.FitnessBeginner, "", false},
	}
	for _, tt := range tests {
		got, err := service.SelectTemplateName(tt.days, tt.level)
		if tt.ok && err != nil {
			t.Fatalf("SelectTemplateName(%d,%s) unexpected error: %v", tt.days, tt.level, err)
		}
		if !tt.ok && err == nil {
			t.Fatalf("SelectTemplateName(%d,%s) expected error", tt.days, tt.level)
		}
		if tt.ok && got != tt.want {
			t.Fatalf("SelectTemplateName(%d,%s) = %q, want %q", tt.days, tt.level, got, tt.want)
		}
	}
}

func TestGenerateTrainingProgramStrength(t *testing.T) {
	users := &fakeUserIdentity{userID: "user-1"}
	inputs := &fakeTrainingInputReader{
		input: domain.TrainingInput{
			UserID:      "user-1",
			Level:       domain.FitnessIntermediate,
			Goal:        domain.GoalStrength,
			DaysPerWeek: 2,
			Injuries:    []string{"shoulder"},
		},
		ok: true,
	}
	templates := &fakeTemplateStorage{
		template: domain.ProgramTemplate{
			ID:   "tpl-1",
			Name: "full_body",
			Structure: domain.TemplateStructure{
				Days: []domain.TemplateDay{
					{Name: "Day 1", Type: "push", MuscleGroups: []string{"chest", "shoulders", "triceps"}},
					{Name: "Day 2", Type: "pull", MuscleGroups: []string{"back", "biceps"}},
				},
			},
		},
		ok: true,
	}
	exercises := &fakeExerciseStorage{
		items: []domain.Exercise{
			{ID: "e1", Name: "Bench Press", MuscleGroup: "chest", Priority: "main", Contraindications: []string{"shoulder"}},
			{ID: "e2", Name: "Machine Press", MuscleGroup: "chest", Priority: "main"},
			{ID: "e3", Name: "Shoulder Press", MuscleGroup: "shoulders", Priority: "secondary"},
			{ID: "e4", Name: "Triceps Pushdown", MuscleGroup: "triceps", Priority: "accessory"},
			{ID: "e5", Name: "Lateral Raise", MuscleGroup: "shoulders", Priority: "accessory"},
			{ID: "e6", Name: "Row", MuscleGroup: "back", Priority: "main"},
			{ID: "e7", Name: "Pulldown", MuscleGroup: "back", Priority: "secondary"},
			{ID: "e8", Name: "Curl", MuscleGroup: "biceps", Priority: "accessory"},
			{ID: "p1", Name: "Band External Rotation", MuscleGroup: "shoulders", Priority: "accessory", PrehabTarget: "shoulder"},
		},
	}
	periods := &fakePeriodizationStorage{
		period: domain.Periodization{
			Week:       1,
			Intensity:  "heavy",
			Percent1RM: "85-90%",
			Reps:       "3-5",
			Rest:       "150-180s",
		},
		ok: true,
	}
	programs := &fakeUserProgramStorage{}

	svc := service.NewTrainingProgramService(users, inputs, templates, exercises, periods, programs)
	svcNow := time.Date(2024, 1, 2, 15, 0, 0, 0, time.UTC)
	svc.SetNowFunc(func() time.Time { return svcNow })

	_, gen, err := svc.Generate(context.Background(), 42)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if len(gen.Days) != 2 {
		t.Fatalf("expected 2 days, got %d", len(gen.Days))
	}
	if len(gen.Days[0].Exercises) == 0 {
		t.Fatal("expected exercises for day 1")
	}
	hasPercent := false
	for _, ex := range gen.Days[0].Exercises {
		if ex.Percent1RM != "" {
			hasPercent = true
		}
		if ex.Name == "Bench Press" {
			t.Fatal("expected contraindicated exercise to be filtered out")
		}
	}
	if !hasPercent {
		t.Fatal("expected percent_1rm for strength goal")
	}
}

type fakeUserIdentity struct {
	userID string
	err    error
}

func (f *fakeUserIdentity) EnsureByTelegramChatID(ctx context.Context, chatID int64) (string, error) {
	return f.userID, f.err
}

type fakeTrainingInputReader struct {
	input domain.TrainingInput
	ok    bool
	err   error
}

func (f *fakeTrainingInputReader) GetByUserID(ctx context.Context, userID string) (domain.TrainingInput, bool, error) {
	return f.input, f.ok, f.err
}

type fakeTemplateStorage struct {
	template domain.ProgramTemplate
	ok       bool
	err      error
}

func (f *fakeTemplateStorage) GetByName(ctx context.Context, name string) (domain.ProgramTemplate, bool, error) {
	return f.template, f.ok, f.err
}

type fakeExerciseStorage struct {
	items []domain.Exercise
}

func (f *fakeExerciseStorage) ListByMuscleGroups(ctx context.Context, groups []string, level domain.FitnessLevel, injuries []string) ([]domain.Exercise, error) {
	out := make([]domain.Exercise, 0)
	for _, ex := range f.items {
		if !contains(groups, ex.MuscleGroup) {
			continue
		}
		if overlaps(injuries, ex.Contraindications) {
			continue
		}
		out = append(out, ex)
	}
	return out, nil
}

func (f *fakeExerciseStorage) ListPrehabByTargets(ctx context.Context, targets []string, level domain.FitnessLevel, injuries []string) ([]domain.Exercise, error) {
	out := make([]domain.Exercise, 0)
	for _, ex := range f.items {
		if ex.PrehabTarget == "" || !contains(targets, ex.PrehabTarget) {
			continue
		}
		out = append(out, ex)
	}
	return out, nil
}

type fakePeriodizationStorage struct {
	period domain.Periodization
	ok     bool
	err    error
}

func (f *fakePeriodizationStorage) GetByWeek(ctx context.Context, week int) (domain.Periodization, bool, error) {
	return f.period, f.ok, f.err
}

type fakeUserProgramStorage struct {
	last domain.UserProgram
	err  error
}

func (f *fakeUserProgramStorage) Insert(ctx context.Context, program domain.UserProgram) (domain.UserProgram, error) {
	if f.err != nil {
		return domain.UserProgram{}, f.err
	}
	f.last = program
	program.ID = "prog-1"
	return program, nil
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func overlaps(a, b []string) bool {
	for _, item := range a {
		for _, cand := range b {
			if item == cand {
				return true
			}
		}
	}
	return false
}
