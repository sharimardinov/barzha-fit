package tests

import (
	"context"
	"encoding/json"
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
					{Name: "Day 1", Type: "push", MuscleGroups: []string{"chest", "shoulders_front", "shoulders_side", "triceps"}},
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
			{ID: "e3", Name: "Shoulder Press", MuscleGroup: "shoulders_front", Priority: "secondary"},
			{ID: "e4", Name: "Triceps Pushdown", MuscleGroup: "triceps", Priority: "accessory"},
			{ID: "e5", Name: "Lateral Raise", MuscleGroup: "shoulders_side", Priority: "accessory"},
			{ID: "e6", Name: "Row", MuscleGroup: "back", Priority: "main"},
			{ID: "e7", Name: "Pulldown", MuscleGroup: "back", Priority: "secondary"},
			{ID: "e8", Name: "Curl", MuscleGroup: "biceps", Priority: "accessory"},
			{ID: "p1", Name: "Band External Rotation", MuscleGroup: "shoulders_rear", Priority: "accessory", PrehabTarget: "shoulder"},
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

func TestContinueProgramWeek2KeepsExercises(t *testing.T) {
	users := &fakeUserIdentity{userID: "user-1"}
	inputs := &fakeTrainingInputReader{
		input: domain.TrainingInput{
			UserID:      "user-1",
			Level:       domain.FitnessBeginner,
			Goal:        domain.GoalHypertrophy,
			DaysPerWeek: 1,
		},
		ok: true,
	}
	templates := &fakeTemplateStorage{}
	exercises := &fakeExerciseStorage{}
	periods := &fakePeriodizationStorage{
		period: domain.Periodization{
			Week:       2,
			Intensity:  "medium",
			Percent1RM: "60-70%",
			Reps:       "10-12",
			Rest:       "2:00-2:30",
		},
		ok: true,
	}

	existingProgram := domain.GeneratedProgram{
		Template: "full_body",
		Week:     1,
		Days: []domain.GeneratedDay{
			{
				Day:   1,
				Name:  "Day 1",
				Focus: "push",
				Type:  "train",
				Exercises: []domain.GeneratedExercise{
					{
						ExerciseID:  "e1",
						Name:        "Bench Press",
						MuscleGroup: "chest",
						Priority:    "main",
						Sets:        4,
						Reps:        "8-12",
						RPE:         "7-8",
						Rest:        "90-120s",
					},
				},
			},
		},
	}
	raw, _ := json.Marshal(existingProgram)
	programs := &fakeUserProgramStorage{
		getOK: true,
		existing: domain.UserProgram{
			ID:            "prog-1",
			UserID:        "user-1",
			TemplateID:    "tpl-1",
			CurrentWeek:   1,
			DaysGenerated: raw,
		},
	}

	svc := service.NewTrainingProgramService(users, inputs, templates, exercises, periods, programs)
	updated, gen, err := svc.Generate(context.Background(), 42)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if updated.CurrentWeek != 2 {
		t.Fatalf("expected current_week 2, got %d", updated.CurrentWeek)
	}
	if gen.Week != 2 {
		t.Fatalf("expected program week 2, got %d", gen.Week)
	}
	if gen.Days[0].Exercises[0].Name != "Bench Press" {
		t.Fatalf("expected same exercise, got %q", gen.Days[0].Exercises[0].Name)
	}
}

func TestAntiAdaptationSubstitution(t *testing.T) {
	users := &fakeUserIdentity{userID: "user-1"}
	inputs := &fakeTrainingInputReader{
		input: domain.TrainingInput{
			UserID:      "user-1",
			Level:       domain.FitnessIntermediate,
			Goal:        domain.GoalStrength,
			DaysPerWeek: 2,
		},
		ok: true,
	}
	templates := &fakeTemplateStorage{
		template: domain.ProgramTemplate{
			ID:   "tpl-1",
			Name: "full_body",
			Structure: domain.TemplateStructure{
				Days: []domain.TemplateDay{
					{Name: "Day 1", Type: "push", MuscleGroups: []string{"chest"}},
					{Name: "Day 2", Type: "push", MuscleGroups: []string{"chest"}},
				},
			},
		},
		ok: true,
	}
	exercises := &fakeExerciseStorage{
		items: []domain.Exercise{
			{ID: "e1", Name: "Bench Press", MuscleGroup: "chest", Priority: "main"},
			{ID: "e2", Name: "Machine Press", MuscleGroup: "chest", Priority: "main", SubstituteFor: []string{"Bench Press"}},
			{ID: "e3", Name: "Incline Press", MuscleGroup: "chest", Priority: "secondary"},
			{ID: "e4", Name: "Chest Press", MuscleGroup: "chest", Priority: "secondary"},
			{ID: "e5", Name: "Cable Fly", MuscleGroup: "chest", Priority: "accessory"},
			{ID: "e6", Name: "Pec Deck", MuscleGroup: "chest", Priority: "accessory"},
		},
	}
	periods := &fakePeriodizationStorage{
		period: domain.Periodization{
			Week:       1,
			Intensity:  "light",
			Percent1RM: "45-50%",
			Reps:       "20-25",
			Rest:       "1:00-1:30",
		},
		ok: true,
	}

	prevProgram := domain.GeneratedProgram{
		Template: "full_body",
		Week:     3,
		Days: []domain.GeneratedDay{
			{
				Day:   1,
				Name:  "Day 1",
				Focus: "push",
				Type:  "train",
				Exercises: []domain.GeneratedExercise{
					{
						ExerciseID:  "e1",
						Name:        "Bench Press",
						MuscleGroup: "chest",
						Priority:    "main",
						Sets:        4,
						Reps:        "3-5",
						RPE:         "8-9",
						Rest:        "150-180s",
					},
				},
			},
			{
				Day:   2,
				Name:  "Day 2",
				Focus: "push",
				Type:  "train",
				Exercises: []domain.GeneratedExercise{
					{
						ExerciseID:  "e1",
						Name:        "Bench Press",
						MuscleGroup: "chest",
						Priority:    "main",
						Sets:        4,
						Reps:        "3-5",
						RPE:         "8-9",
						Rest:        "150-180s",
					},
				},
			},
		},
	}
	raw, _ := json.Marshal(prevProgram)
	programs := &fakeUserProgramStorage{
		getOK: true,
		existing: domain.UserProgram{
			ID:            "prog-1",
			UserID:        "user-1",
			TemplateID:    "tpl-1",
			CurrentWeek:   3,
			DaysGenerated: raw,
		},
	}

	svc := service.NewTrainingProgramService(users, inputs, templates, exercises, periods, programs)
	_, gen, err := svc.Generate(context.Background(), 42)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if gen.Days[0].Exercises[0].Name != "Machine Press" {
		t.Fatalf("expected substitute, got %q", gen.Days[0].Exercises[0].Name)
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

func (f *fakeExerciseStorage) ListSubstitutes(ctx context.Context, names []string, level domain.FitnessLevel, injuries []string) ([]domain.Exercise, error) {
	out := make([]domain.Exercise, 0)
	for _, ex := range f.items {
		if len(ex.SubstituteFor) == 0 {
			continue
		}
		if overlaps(names, ex.SubstituteFor) {
			out = append(out, ex)
		}
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
	last     domain.UserProgram
	existing domain.UserProgram
	getOK    bool
	err      error
}

func (f *fakeUserProgramStorage) GetLatestByUserID(ctx context.Context, userID string) (domain.UserProgram, bool, error) {
	if f.err != nil {
		return domain.UserProgram{}, false, f.err
	}
	if !f.getOK {
		return domain.UserProgram{}, false, nil
	}
	return f.existing, true, nil
}

func (f *fakeUserProgramStorage) Insert(ctx context.Context, program domain.UserProgram) (domain.UserProgram, error) {
	if f.err != nil {
		return domain.UserProgram{}, f.err
	}
	f.last = program
	program.ID = "prog-1"
	return program, nil
}

func (f *fakeUserProgramStorage) Update(ctx context.Context, program domain.UserProgram) (domain.UserProgram, error) {
	if f.err != nil {
		return domain.UserProgram{}, f.err
	}
	f.last = program
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
