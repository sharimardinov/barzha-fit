package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"barzhafit/backend/domain"
)

type TrainingInputReader interface {
	GetByUserID(ctx context.Context, userID string) (domain.TrainingInput, bool, error)
}

type ProgramTemplateStorage interface {
	GetByName(ctx context.Context, name string) (domain.ProgramTemplate, bool, error)
}

type ExerciseStorage interface {
	ListByMuscleGroups(ctx context.Context, groups []string, level domain.FitnessLevel, injuries []string) ([]domain.Exercise, error)
	ListPrehabByTargets(ctx context.Context, targets []string, level domain.FitnessLevel, injuries []string) ([]domain.Exercise, error)
	ListSubstitutes(ctx context.Context, names []string, level domain.FitnessLevel, injuries []string) ([]domain.Exercise, error)
}

type PeriodizationStorage interface {
	GetByWeek(ctx context.Context, week int) (domain.Periodization, bool, error)
}

type UserProgramStorage interface {
	Insert(ctx context.Context, program domain.UserProgram) (domain.UserProgram, error)
	Update(ctx context.Context, program domain.UserProgram) (domain.UserProgram, error)
	GetLatestByUserID(ctx context.Context, userID string) (domain.UserProgram, bool, error)
}

type TrainingProgramService struct {
	users         UserIdentityStorage
	inputs        TrainingInputReader
	templates     ProgramTemplateStorage
	exercises     ExerciseStorage
	periodization PeriodizationStorage
	programs      UserProgramStorage
	now           func() time.Time
}

func NewTrainingProgramService(
	users UserIdentityStorage,
	inputs TrainingInputReader,
	templates ProgramTemplateStorage,
	exercises ExerciseStorage,
	periodization PeriodizationStorage,
	programs UserProgramStorage,
) *TrainingProgramService {
	return &TrainingProgramService{
		users:         users,
		inputs:        inputs,
		templates:     templates,
		exercises:     exercises,
		periodization: periodization,
		programs:      programs,
		now:           time.Now,
	}
}

func (s *TrainingProgramService) SetNowFunc(fn func() time.Time) {
	if fn != nil {
		s.now = fn
	}
}

func SelectTemplateName(days int, level domain.FitnessLevel) (string, error) {
	switch days {
	case 2:
		return "full_body", nil
	case 3:
		if level == domain.FitnessBeginner {
			return "full_body", nil
		}
		return "push_pull_legs", nil
	case 4:
		return "upper_lower_x2", nil
	case 5:
		return "upper_lower_arm_day", nil
	case 6:
		return "ppl_x2", nil
	default:
		return "", fmt.Errorf("unsupported days_per_week")
	}
}

func (s *TrainingProgramService) Generate(ctx context.Context, chatID int64) (domain.UserProgram, domain.GeneratedProgram, error) {
	userID, err := s.users.EnsureByTelegramChatID(ctx, chatID)
	if err != nil {
		return domain.UserProgram{}, domain.GeneratedProgram{}, err
	}
	input, ok, err := s.inputs.GetByUserID(ctx, userID)
	if err != nil {
		return domain.UserProgram{}, domain.GeneratedProgram{}, err
	}
	if !ok {
		return domain.UserProgram{}, domain.GeneratedProgram{}, fmt.Errorf("training_input_not_found")
	}

	existing, has, err := s.programs.GetLatestByUserID(ctx, userID)
	if err != nil {
		return domain.UserProgram{}, domain.GeneratedProgram{}, err
	}

	if has && existing.CurrentWeek < 3 {
		nextWeek := existing.CurrentWeek + 1
		period := s.resolvePeriodization(ctx, nextWeek)
		prev, err := decodeGeneratedProgram(existing.DaysGenerated)
		if err != nil {
			return domain.UserProgram{}, domain.GeneratedProgram{}, err
		}
		updatedProgram := applyPeriodization(prev, input.Goal, period)
		raw, err := json.Marshal(updatedProgram)
		if err != nil {
			return domain.UserProgram{}, domain.GeneratedProgram{}, err
		}
		existing.CurrentWeek = nextWeek
		existing.DaysGenerated = raw
		updated, err := s.programs.Update(ctx, existing)
		if err != nil {
			return domain.UserProgram{}, domain.GeneratedProgram{}, err
		}
		return updated, updatedProgram, nil
	}

	templateName, err := SelectTemplateName(input.DaysPerWeek, input.Level)
	if err != nil {
		return domain.UserProgram{}, domain.GeneratedProgram{}, err
	}
	template, ok, err := s.templates.GetByName(ctx, templateName)
	if err != nil {
		return domain.UserProgram{}, domain.GeneratedProgram{}, err
	}
	if !ok {
		return domain.UserProgram{}, domain.GeneratedProgram{}, fmt.Errorf("template_not_found")
	}

	days := template.Structure.Days
	if len(days) < input.DaysPerWeek {
		return domain.UserProgram{}, domain.GeneratedProgram{}, fmt.Errorf("template_days_mismatch")
	}
	if len(days) > input.DaysPerWeek {
		days = days[:input.DaysPerWeek]
	}

	week := 1
	period := s.resolvePeriodization(ctx, week)
	avoidNames := map[string]bool{}
	avoidList := []string{}
	if has {
		prev, err := decodeGeneratedProgram(existing.DaysGenerated)
		if err != nil {
			return domain.UserProgram{}, domain.GeneratedProgram{}, err
		}
		avoidNames, avoidList = collectExerciseNames(prev)
	}
	substitutes, err := s.fetchSubstitutes(ctx, avoidList, input)
	if err != nil {
		return domain.UserProgram{}, domain.GeneratedProgram{}, err
	}

	gen := domain.GeneratedProgram{
		Template:      template.Name,
		Week:          week,
		Periodization: period,
		Days:          make([]domain.GeneratedDay, 0, len(days)),
	}
	for i, day := range days {
		plan, err := s.generateDay(ctx, i+1, day, input, period, avoidNames, substitutes)
		if err != nil {
			return domain.UserProgram{}, domain.GeneratedProgram{}, err
		}
		gen.Days = append(gen.Days, plan)
	}

	raw, err := json.Marshal(gen)
	if err != nil {
		return domain.UserProgram{}, domain.GeneratedProgram{}, err
	}
	startDate := s.now().UTC().Truncate(24 * time.Hour)
	if has {
		existing.CurrentWeek = week
		existing.DaysGenerated = raw
		updated, err := s.programs.Update(ctx, existing)
		if err != nil {
			return domain.UserProgram{}, domain.GeneratedProgram{}, err
		}
		return updated, gen, nil
	}
	inserted, err := s.programs.Insert(ctx, domain.UserProgram{
		UserID:        userID,
		TemplateID:    template.ID,
		StartDate:     startDate,
		CurrentWeek:   week,
		DaysGenerated: raw,
	})
	if err != nil {
		return domain.UserProgram{}, domain.GeneratedProgram{}, err
	}
	return inserted, gen, nil
}

func (s *TrainingProgramService) generateDay(
	ctx context.Context,
	dayIndex int,
	day domain.TemplateDay,
	input domain.TrainingInput,
	period domain.Periodization,
	avoidNames map[string]bool,
	substitutes map[string][]domain.Exercise,
) (domain.GeneratedDay, error) {
	groups := normalizeGroups(day.MuscleGroups)
	exercises, err := s.exercises.ListByMuscleGroups(ctx, groups, input.Level, input.Injuries)
	if err != nil {
		return domain.GeneratedDay{}, err
	}
	grouped := groupByPriority(exercises)
	used := map[string]bool{}

	selected := make([]domain.Exercise, 0, 6)
	selected = append(selected, pickByPriority(grouped, "main", groups, 1, used)...)
	selected = append(selected, pickByPriority(grouped, "secondary", groups, 2, used)...)
	selected = append(selected, pickByPriority(grouped, "accessory", groups, 2, used)...)

	if len(selected) < 5 {
		selected = append(selected, pickFallback(grouped, groups, 5-len(selected), used)...)
	}

	if len(avoidNames) > 0 {
		selected = replaceWithSubstitutes(selected, substitutes, avoidNames, used)
	}

	if len(input.Injuries) > 0 {
		prehab, err := s.exercises.ListPrehabByTargets(ctx, input.Injuries, input.Level, input.Injuries)
		if err != nil {
			return domain.GeneratedDay{}, err
		}
		for _, ex := range prehab {
			if !used[ex.ID] {
				selected = append(selected, ex)
				used[ex.ID] = true
				break
			}
		}
	}

	items := make([]domain.GeneratedExercise, 0, len(selected))
	for _, ex := range selected {
		sets, reps, rpe, rest, percent := buildPrescription(input.Goal, ex.Priority, period)
		if ex.PrehabTarget != "" && len(input.Injuries) > 0 {
			sets, reps, rpe, rest, percent = prehabPrescription()
		}
		tags := []string{}
		if period.Week == 3 {
			tags = []string{"antiadaptation"}
		}
		items = append(items, domain.GeneratedExercise{
			ExerciseID:  ex.ID,
			Name:        ex.Name,
			MuscleGroup: ex.MuscleGroup,
			Priority:    ex.Priority,
			Sets:        sets,
			Reps:        reps,
			RPE:         rpe,
			Rest:        rest,
			Percent1RM:  percent,
			Tags:        tags,
		})
	}

	focus := day.Type
	if focus == "" {
		focus = strings.Join(groups, "/")
	}
	name := day.Name
	if name == "" {
		name = fmt.Sprintf("Day %d", dayIndex)
	}
	return domain.GeneratedDay{
		Day:       dayIndex,
		Name:      name,
		Focus:     focus,
		Type:      "train",
		Exercises: items,
	}, nil
}

func normalizeGroups(groups []string) []string {
	out := make([]string, 0, len(groups))
	seen := map[string]bool{}
	add := func(value string) {
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		out = append(out, value)
	}
	for _, g := range groups {
		v := strings.TrimSpace(strings.ToLower(g))
		if v == "" {
			continue
		}
		switch v {
		case "shoulders", "shoulder":
			add("shoulders_front")
			add("shoulders_side")
			add("shoulders_rear")
		case "legs":
			add("quads")
			add("hamstrings")
			add("glutes")
		case "arms":
			add("biceps")
			add("triceps")
		default:
			add(v)
		}
	}
	return out
}

func groupByPriority(items []domain.Exercise) map[string]map[string][]domain.Exercise {
	out := map[string]map[string][]domain.Exercise{}
	for _, item := range items {
		p := strings.ToLower(strings.TrimSpace(item.Priority))
		if p == "" {
			p = "accessory"
		}
		if _, ok := out[p]; !ok {
			out[p] = map[string][]domain.Exercise{}
		}
		group := strings.ToLower(strings.TrimSpace(item.MuscleGroup))
		out[p][group] = append(out[p][group], item)
	}
	for _, byGroup := range out {
		for k := range byGroup {
			sort.Slice(byGroup[k], func(i, j int) bool {
				return byGroup[k][i].Name < byGroup[k][j].Name
			})
		}
	}
	return out
}

func pickByPriority(grouped map[string]map[string][]domain.Exercise, priority string, groups []string, count int, used map[string]bool) []domain.Exercise {
	if count <= 0 {
		return nil
	}
	byGroup := grouped[priority]
	if len(byGroup) == 0 {
		return nil
	}
	selected := make([]domain.Exercise, 0, count)
	index := map[string]int{}
	for len(selected) < count {
		added := false
		for _, group := range groups {
			list := byGroup[group]
			i := index[group]
			for i < len(list) && used[list[i].ID] {
				i++
			}
			if i < len(list) {
				ex := list[i]
				index[group] = i + 1
				used[ex.ID] = true
				selected = append(selected, ex)
				added = true
				if len(selected) >= count {
					break
				}
			}
		}
		if !added {
			break
		}
	}
	return selected
}

func pickFallback(grouped map[string]map[string][]domain.Exercise, groups []string, count int, used map[string]bool) []domain.Exercise {
	if count <= 0 {
		return nil
	}
	order := []string{"main", "secondary", "accessory"}
	selected := make([]domain.Exercise, 0, count)
	for _, pr := range order {
		byGroup := grouped[pr]
		if len(byGroup) == 0 {
			continue
		}
		for _, group := range groups {
			for _, ex := range byGroup[group] {
				if used[ex.ID] {
					continue
				}
				used[ex.ID] = true
				selected = append(selected, ex)
				if len(selected) >= count {
					return selected
				}
			}
		}
	}
	return selected
}

func buildPrescription(goal domain.FitnessGoal, priority string, period domain.Periodization) (int, string, string, string, string) {
	sets := 3
	switch priority {
	case "main":
		sets = 4
	case "secondary":
		sets = 3
	case "accessory":
		sets = 2
	}

	reps := goalReps(goal, period)
	rpe := intensityRPE(period.Intensity, goal)
	rest := period.Rest
	if rest == "" {
		rest = goalRest(goal)
	}
	percent := ""
	if goal == domain.GoalStrength {
		if period.Percent1RM != "" {
			percent = period.Percent1RM
		} else {
			percent = "80-90%"
		}
	}
	return sets, reps, rpe, rest, percent
}

func goalReps(goal domain.FitnessGoal, period domain.Periodization) string {
	switch goal {
	case domain.GoalStrength:
		if period.Reps != "" {
			return period.Reps
		}
		return "3-6"
	case domain.GoalHypertrophy:
		return "8-12"
	case domain.GoalFatLoss:
		return "10-15"
	default:
		return "8-12"
	}
}

func goalRest(goal domain.FitnessGoal) string {
	switch goal {
	case domain.GoalStrength:
		return "120-180s"
	case domain.GoalHypertrophy:
		return "90-120s"
	case domain.GoalFatLoss:
		return "60-90s"
	default:
		return "90-120s"
	}
}

func intensityRPE(intensity string, goal domain.FitnessGoal) string {
	switch strings.ToLower(strings.TrimSpace(intensity)) {
	case "light":
		return "6-7"
	case "medium":
		return "7-8"
	case "heavy":
		return "8-9"
	default:
		switch goal {
		case domain.GoalStrength:
			return "8-9"
		case domain.GoalFatLoss:
			return "6-7"
		default:
			return "7-8"
		}
	}
}

func prehabPrescription() (int, string, string, string, string) {
	return 2, "12-15", "6-7", "60-90s", ""
}

func decodeGeneratedProgram(raw json.RawMessage) (domain.GeneratedProgram, error) {
	if len(raw) == 0 {
		return domain.GeneratedProgram{}, fmt.Errorf("empty program payload")
	}
	var out domain.GeneratedProgram
	if err := json.Unmarshal(raw, &out); err != nil {
		return domain.GeneratedProgram{}, err
	}
	return out, nil
}

func applyPeriodization(program domain.GeneratedProgram, goal domain.FitnessGoal, period domain.Periodization) domain.GeneratedProgram {
	program.Week = period.Week
	program.Periodization = period
	for i := range program.Days {
		day := &program.Days[i]
		for j := range day.Exercises {
			ex := &day.Exercises[j]
			sets, reps, rpe, rest, percent := buildPrescription(goal, ex.Priority, period)
			ex.Sets = sets
			ex.Reps = reps
			ex.RPE = rpe
			ex.Rest = rest
			ex.Percent1RM = percent
			ex.Tags = nil
			if period.Week == 3 {
				ex.Tags = []string{"antiadaptation"}
			}
		}
	}
	return program
}

func (s *TrainingProgramService) resolvePeriodization(ctx context.Context, week int) domain.Periodization {
	period, ok, err := s.periodization.GetByWeek(ctx, week)
	if err != nil || !ok {
		fallback := fallbackPeriodization()
		fallback.Week = week
		return fallback
	}
	return period
}

func collectExerciseNames(program domain.GeneratedProgram) (map[string]bool, []string) {
	out := map[string]bool{}
	list := make([]string, 0)
	seenOriginal := map[string]bool{}
	for _, day := range program.Days {
		for _, ex := range day.Exercises {
			key := normalizeName(ex.Name)
			if key == "" {
				continue
			}
			out[key] = true
			if !seenOriginal[ex.Name] {
				seenOriginal[ex.Name] = true
				list = append(list, ex.Name)
			}
		}
	}
	return out, list
}

func (s *TrainingProgramService) fetchSubstitutes(ctx context.Context, names []string, input domain.TrainingInput) (map[string][]domain.Exercise, error) {
	if len(names) == 0 {
		return map[string][]domain.Exercise{}, nil
	}
	items, err := s.exercises.ListSubstitutes(ctx, names, input.Level, input.Injuries)
	if err != nil {
		return nil, err
	}
	out := map[string][]domain.Exercise{}
	for _, item := range items {
		for _, target := range item.SubstituteFor {
			key := normalizeName(target)
			if key == "" {
				continue
			}
			out[key] = append(out[key], item)
		}
	}
	for key := range out {
		sort.Slice(out[key], func(i, j int) bool {
			return out[key][i].Name < out[key][j].Name
		})
	}
	return out, nil
}

func replaceWithSubstitutes(selected []domain.Exercise, substitutes map[string][]domain.Exercise, avoidNames map[string]bool, used map[string]bool) []domain.Exercise {
	if len(selected) == 0 || len(avoidNames) == 0 {
		return selected
	}
	out := make([]domain.Exercise, 0, len(selected))
	for _, ex := range selected {
		key := normalizeName(ex.Name)
		if !avoidNames[key] {
			out = append(out, ex)
			continue
		}
		candidates := substitutes[key]
		if replacement, ok := pickSubstitute(candidates, ex, used); ok {
			used[replacement.ID] = true
			out = append(out, replacement)
			continue
		}
		out = append(out, ex)
	}
	return out
}

func pickSubstitute(candidates []domain.Exercise, original domain.Exercise, used map[string]bool) (domain.Exercise, bool) {
	if len(candidates) == 0 {
		return domain.Exercise{}, false
	}
	firstMatch := func(matchGroup bool) (domain.Exercise, bool) {
		for _, cand := range candidates {
			if used[cand.ID] {
				continue
			}
			if matchGroup && cand.MuscleGroup != original.MuscleGroup {
				continue
			}
			if cand.Priority != "" && original.Priority != "" && cand.Priority != original.Priority {
				continue
			}
			return cand, true
		}
		return domain.Exercise{}, false
	}
	if ex, ok := firstMatch(true); ok {
		return ex, true
	}
	return firstMatch(false)
}

func normalizeName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func fallbackPeriodization() domain.Periodization {
	return domain.Periodization{
		Week:       1,
		Intensity:  "light",
		Percent1RM: "45-50%",
		Reps:       "20-25",
		Rest:       "1:00-1:30",
	}
}
