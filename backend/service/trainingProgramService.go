package service

import (
	"context"
	"encoding/json"
	"fmt"

	// "math/rand"
	"sort"
	"strings"
	"time"

	"barzhafit/backend/domain"
)

// -------------------- INTERFACES --------------------

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

type UserIdentityStorage interface {
	EnsureByTelegramChatID(ctx context.Context, chatID int64) (string, error)
}

// -------------------- SERVICE --------------------

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

// -------------------- WEEK STATE --------------------

type WeekState struct {
	UsedExerciseIDs map[string]bool
	UsedNames       map[string]bool
	UsedByGroup     map[string]int
	UsedByPattern   map[string]int
}

func newWeekState() *WeekState {
	return &WeekState{
		UsedExerciseIDs: map[string]bool{},
		UsedNames:       map[string]bool{},
		UsedByGroup:     map[string]int{},
		UsedByPattern:   map[string]int{},
	}
}

// -------------------- DAY CAPS --------------------

var dayCaps = map[string]int{
	"chest":           1,
	"back":            2,
	"quads":           1,
	"hamstrings":      1,
	"glutes":          1,
	"shoulders_front": 1,
	"shoulders_side":  1,
	"shoulders_rear":  1,
	"biceps":          1,
	"triceps":         1,
}

type DayState struct {
	UsedByGroup map[string]int
	UsedIDs     map[string]bool
}

func newDayState() *DayState {
	return &DayState{
		UsedByGroup: map[string]int{},
		UsedIDs:     map[string]bool{},
	}
}

func (ds *DayState) canUse(ex domain.Exercise, ws *WeekState) bool {
	if ds.UsedIDs[ex.ID] {
		return false
	}
	if ws.UsedExerciseIDs[ex.ID] {
		return false
	}
	key := normalizeName(ex.Name)
	if ws.UsedNames[key] {
		return false
	}
	group := strings.ToLower(strings.TrimSpace(ex.MuscleGroup))
	if cap, ok := dayCaps[group]; ok {
		if ds.UsedByGroup[group] >= cap {
			return false
		}
	}
	return true
}

func (ds *DayState) markUsed(ex domain.Exercise, ws *WeekState) {
	ds.UsedIDs[ex.ID] = true
	ws.UsedExerciseIDs[ex.ID] = true
	ws.UsedNames[normalizeName(ex.Name)] = true
	group := strings.ToLower(strings.TrimSpace(ex.MuscleGroup))
	ds.UsedByGroup[group]++
	ws.UsedByGroup[group]++
	for _, t := range ex.Type {
		ws.UsedByPattern[strings.ToLower(t)]++
	}
}

// -------------------- EXERCISE HELPERS --------------------

func hasTag(ex domain.Exercise, tag string) bool {
	tag = strings.ToLower(strings.TrimSpace(tag))
	for _, t := range ex.Type {
		if strings.ToLower(strings.TrimSpace(t)) == tag {
			return true
		}
	}
	return false
}

func priorityRank(p string) int {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "main":
		return 3
	case "secondary":
		return 2
	case "accessory":
		return 1
	default:
		return 0
	}
}

// -------------------- SLOT DEFINITIONS --------------------

type Slot struct {
	Name            string
	RequiredGroup   []string
	RequiredPattern []string
	PreferPriority  string
	PreferCompound  bool
}

// FULL BODY (3 days)
func getSlotsFullBody(dayIndex int) []Slot {
	slots := make([]Slot, 6)

	// Rotate lower patterns: Day1=squat, Day2=hinge, Day3=squat
	lowerPattern := "squat"
	lowerGroups := []string{"quads", "glutes"}
	if dayIndex == 2 {
		lowerPattern = "hinge"
		lowerGroups = []string{"hamstrings", "glutes"}
	}

	slots[0] = Slot{
		Name:            "lower_primary",
		RequiredGroup:   lowerGroups,
		RequiredPattern: []string{lowerPattern, "lower", "compound"},
		PreferPriority:  "main",
		PreferCompound:  true,
	}

	// Rotate push: Day1=chest, Day2=chest, Day3=shoulders_front
	pushGroup := "chest"
	if dayIndex == 3 {
		pushGroup = "shoulders_front"
	}
	slots[1] = Slot{
		Name:            "push_primary",
		RequiredGroup:   []string{pushGroup},
		RequiredPattern: []string{"push", "compound"},
		PreferPriority:  "main",
		PreferCompound:  true,
	}

	slots[2] = Slot{
		Name:            "pull_primary",
		RequiredGroup:   []string{"back"},
		RequiredPattern: []string{"pull", "compound"},
		PreferPriority:  "main",
		PreferCompound:  true,
	}

	slots[3] = Slot{
		Name:            "lower_secondary",
		RequiredGroup:   []string{"hamstrings", "glutes"},
		RequiredPattern: []string{"isolation", "lower"},
		PreferPriority:  "secondary",
		PreferCompound:  false,
	}

	slots[4] = Slot{
		Name:            "delts",
		RequiredGroup:   []string{"shoulders_side", "shoulders_rear"},
		RequiredPattern: []string{"isolation"},
		PreferPriority:  "accessory",
		PreferCompound:  false,
	}

	armGroup := "biceps"
	if dayIndex%2 == 0 {
		armGroup = "triceps"
	}
	slots[5] = Slot{
		Name:            "arms",
		RequiredGroup:   []string{armGroup},
		RequiredPattern: []string{"isolation"},
		PreferPriority:  "accessory",
		PreferCompound:  false,
	}

	return slots
}

// PUSH DAY
func getSlotsPush() []Slot {
	return []Slot{
		{
			Name:            "chest_main",
			RequiredGroup:   []string{"chest"},
			RequiredPattern: []string{"push", "compound"},
			PreferPriority:  "main",
			PreferCompound:  true,
		},
		{
			Name:            "shoulders_main",
			RequiredGroup:   []string{"shoulders_front"},
			RequiredPattern: []string{"push", "compound"},
			PreferPriority:  "main",
			PreferCompound:  true,
		},
		{
			Name:            "chest_secondary",
			RequiredGroup:   []string{"chest"},
			RequiredPattern: []string{"push"},
			PreferPriority:  "secondary",
			PreferCompound:  false,
		},
		{
			Name:            "shoulders_side",
			RequiredGroup:   []string{"shoulders_side"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "accessory",
			PreferCompound:  false,
		},
		{
			Name:            "triceps_1",
			RequiredGroup:   []string{"triceps"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "secondary",
			PreferCompound:  false,
		},
		{
			Name:            "triceps_2",
			RequiredGroup:   []string{"triceps"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "accessory",
			PreferCompound:  false,
		},
	}
}

// PULL DAY
func getSlotsPull() []Slot {
	return []Slot{
		{
			Name:            "back_vertical",
			RequiredGroup:   []string{"back"},
			RequiredPattern: []string{"pull", "compound"},
			PreferPriority:  "main",
			PreferCompound:  true,
		},
		{
			Name:            "back_horizontal",
			RequiredGroup:   []string{"back"},
			RequiredPattern: []string{"pull", "compound"},
			PreferPriority:  "main",
			PreferCompound:  true,
		},
		{
			Name:            "back_secondary",
			RequiredGroup:   []string{"back"},
			RequiredPattern: []string{"pull"},
			PreferPriority:  "secondary",
			PreferCompound:  false,
		},
		{
			Name:            "shoulders_rear",
			RequiredGroup:   []string{"shoulders_rear"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "accessory",
			PreferCompound:  false,
		},
		{
			Name:            "biceps_1",
			RequiredGroup:   []string{"biceps"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "secondary",
			PreferCompound:  false,
		},
		{
			Name:            "biceps_2",
			RequiredGroup:   []string{"biceps"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "accessory",
			PreferCompound:  false,
		},
	}
}

// LEGS DAY
func getSlotsLegs() []Slot {
	return []Slot{
		{
			Name:            "quads_main",
			RequiredGroup:   []string{"quads"},
			RequiredPattern: []string{"squat", "compound", "lower"},
			PreferPriority:  "main",
			PreferCompound:  true,
		},
		{
			Name:            "hamstrings_main",
			RequiredGroup:   []string{"hamstrings"},
			RequiredPattern: []string{"hinge", "compound", "lower"},
			PreferPriority:  "main",
			PreferCompound:  true,
		},
		{
			Name:            "glutes_main",
			RequiredGroup:   []string{"glutes"},
			RequiredPattern: []string{"compound", "lower"},
			PreferPriority:  "main",
			PreferCompound:  true,
		},
		{
			Name:            "quads_secondary",
			RequiredGroup:   []string{"quads"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "secondary",
			PreferCompound:  false,
		},
		{
			Name:            "hamstrings_secondary",
			RequiredGroup:   []string{"hamstrings"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "secondary",
			PreferCompound:  false,
		},
		{
			Name:            "glutes_accessory",
			RequiredGroup:   []string{"glutes"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "accessory",
			PreferCompound:  false,
		},
	}
}

// UPPER DAY
func getSlotsUpper() []Slot {
	return []Slot{
		{
			Name:            "chest_main",
			RequiredGroup:   []string{"chest"},
			RequiredPattern: []string{"push", "compound"},
			PreferPriority:  "main",
			PreferCompound:  true,
		},
		{
			Name:            "back_main",
			RequiredGroup:   []string{"back"},
			RequiredPattern: []string{"pull", "compound"},
			PreferPriority:  "main",
			PreferCompound:  true,
		},
		{
			Name:            "shoulders_main",
			RequiredGroup:   []string{"shoulders_front"},
			RequiredPattern: []string{"push", "compound"},
			PreferPriority:  "secondary",
			PreferCompound:  true,
		},
		{
			Name:            "back_secondary",
			RequiredGroup:   []string{"back"},
			RequiredPattern: []string{"pull"},
			PreferPriority:  "secondary",
			PreferCompound:  false,
		},
		{
			Name:            "shoulders_accessory",
			RequiredGroup:   []string{"shoulders_side", "shoulders_rear"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "accessory",
			PreferCompound:  false,
		},
		{
			Name:            "arms",
			RequiredGroup:   []string{"biceps", "triceps"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "accessory",
			PreferCompound:  false,
		},
	}
}

// ARMS DAY
func getSlotsArms() []Slot {
	return []Slot{
		{
			Name:            "biceps_1",
			RequiredGroup:   []string{"biceps"},
			RequiredPattern: []string{"compound"},
			PreferPriority:  "main",
			PreferCompound:  true,
		},
		{
			Name:            "triceps_1",
			RequiredGroup:   []string{"triceps"},
			RequiredPattern: []string{"compound"},
			PreferPriority:  "main",
			PreferCompound:  true,
		},
		{
			Name:            "biceps_2",
			RequiredGroup:   []string{"biceps"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "secondary",
			PreferCompound:  false,
		},
		{
			Name:            "triceps_2",
			RequiredGroup:   []string{"triceps"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "secondary",
			PreferCompound:  false,
		},
		{
			Name:            "shoulders_side",
			RequiredGroup:   []string{"shoulders_side"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "accessory",
			PreferCompound:  false,
		},
		{
			Name:            "shoulders_rear",
			RequiredGroup:   []string{"shoulders_rear"},
			RequiredPattern: []string{"isolation"},
			PreferPriority:  "accessory",
			PreferCompound:  false,
		},
	}
}

func getSlotsByType(dayType string, dayIndex int) []Slot {
	switch dayType {
	case "full_body":
		return getSlotsFullBody(dayIndex)
	case "push":
		return getSlotsPush()
	case "pull":
		return getSlotsPull()
	case "legs":
		return getSlotsLegs()
	case "upper":
		return getSlotsUpper()
	case "arms":
		return getSlotsArms()
	default:
		return getSlotsFullBody(dayIndex)
	}
}

// -------------------- SCORING --------------------

func scoreExercise(ex domain.Exercise, slot Slot, ds *DayState, ws *WeekState) int {
	if !ds.canUse(ex, ws) {
		return -9999
	}

	score := 0
	group := strings.ToLower(strings.TrimSpace(ex.MuscleGroup))

	// +3 if priority matches
	if ex.Priority == slot.PreferPriority {
		score += 3
	} else if ex.Priority == "main" {
		score += 2
	} else if ex.Priority == "secondary" {
		score += 1
	}

	// +2 if compound and slot prefers compound
	if slot.PreferCompound && hasTag(ex, "compound") {
		score += 2
	}

	// +2 if pattern matches
	for _, reqPattern := range slot.RequiredPattern {
		if hasTag(ex, reqPattern) {
			score += 2
			break
		}
	}

	// +1 if group matches
	for _, reqGroup := range slot.RequiredGroup {
		if group == strings.ToLower(reqGroup) {
			score += 1
			break
		}
	}

	// -2 if this is already 2nd back in day
	if group == "back" && ds.UsedByGroup["back"] >= 1 {
		score -= 2
	}

	// +1 if this group is underutilized in week
	if ws.UsedByGroup[group] < 2 {
		score += 1
	}

	return score
}

func pickBestExercise(pool []domain.Exercise, slot Slot, ds *DayState, ws *WeekState) (domain.Exercise, bool) {
	type candidate struct {
		ex    domain.Exercise
		score int
	}

	candidates := make([]candidate, 0, len(pool))
	for _, ex := range pool {
		score := scoreExercise(ex, slot, ds, ws)
		if score > -9999 {
			candidates = append(candidates, candidate{ex: ex, score: score})
		}
	}

	if len(candidates) == 0 {
		return domain.Exercise{}, false
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].ex.Name < candidates[j].ex.Name
	})

	return candidates[0].ex, true
}

// -------------------- MAIN GENERATOR --------------------

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

	// Week progression
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

	// Generate new program
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

	week := 1
	period := s.resolvePeriodization(ctx, week)

	// Fetch ALL exercises once
	allGroups := []string{"chest", "back", "quads", "hamstrings", "glutes",
		"shoulders_front", "shoulders_side", "shoulders_rear", "biceps", "triceps"}
	pool, err := s.exercises.ListByMuscleGroups(ctx, allGroups, input.Level, input.Injuries)
	if err != nil {
		return domain.UserProgram{}, domain.GeneratedProgram{}, err
	}

	ws := newWeekState()

	gen := domain.GeneratedProgram{
		Template:      template.Name,
		Week:          week,
		Periodization: period,
		Days:          make([]domain.GeneratedDay, 0, len(template.Structure.Days)),
	}

	for dayIdx, templateDay := range template.Structure.Days {
		day, err := s.generateDayBySlots(dayIdx+1, templateDay, pool, input, period, ws)
		if err != nil {
			return domain.UserProgram{}, domain.GeneratedProgram{}, err
		}
		gen.Days = append(gen.Days, day)
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

func (s *TrainingProgramService) generateDayBySlots(
	dayIndex int,
	templateDay domain.TemplateDay,
	pool []domain.Exercise,
	input domain.TrainingInput,
	period domain.Periodization,
	ws *WeekState,
) (domain.GeneratedDay, error) {
	ds := newDayState()

	dayType := templateDay.Type
	if dayType == "" {
		dayType = "full_body"
	}

	slots := getSlotsByType(dayType, dayIndex)
	selected := make([]domain.Exercise, 0, len(slots))

	for _, slot := range slots {
		ex, ok := pickBestExercise(pool, slot, ds, ws)
		if ok {
			selected = append(selected, ex)
			ds.markUsed(ex, ws)
		}
	}

	// Add prehab if needed
	if len(input.Injuries) > 0 {
		prehab, err := s.exercises.ListPrehabByTargets(context.Background(), input.Injuries, input.Level, input.Injuries)
		if err == nil {
			for _, ex := range prehab {
				if !ds.UsedIDs[ex.ID] && !ws.UsedExerciseIDs[ex.ID] {
					selected = append(selected, ex)
					ds.markUsed(ex, ws)
					break
				}
			}
		}
	}

	// Build prescription
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

	dayName := templateDay.Name
	if dayName == "" {
		dayName = fmt.Sprintf("Day %d", dayIndex)
	}

	return domain.GeneratedDay{
		Day:       dayIndex,
		Name:      dayName,
		Focus:     dayType,
		Type:      "train",
		Exercises: items,
	}, nil
}

// -------------------- HELPERS --------------------

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

func normalizeName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
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

func fallbackPeriodization() domain.Periodization {
	return domain.Periodization{
		Week:       1,
		Intensity:  "light",
		Percent1RM: "45-50%",
		Reps:       "20-25",
		Rest:       "1:00-1:30",
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
