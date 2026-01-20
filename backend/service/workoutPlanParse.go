package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"barzhafit/backend/domain"
)

var (
	workoutSetsRe     = regexp.MustCompile(`(?i)^\s*(\d+)\s*[xх]\s*(\d+)\s*$`)
	workoutDurationRe = regexp.MustCompile(`(?i)^\s*(\d+(?:[\.,]\d+)?)\s*(мин|min|m)\s*$`)
	workoutNumberRe   = regexp.MustCompile(`[-+]?\d+(?:[\.,]\d+)?`)
	workoutHeaderRe   = regexp.MustCompile(`(?i)^\s*(day|день)\s*\d+\b`)
	workoutNumberOnly = regexp.MustCompile(`^\s*\d+(?:[.,]\d+)?\s*$`)
	slashFieldRe      = regexp.MustCompile(`\s*/\s*`)
)

func BuildWorkoutPlanFromText(text string) (domain.WorkoutPlan, []string) {
	plan := domain.WorkoutPlan{DefaultRestSec: defaultWorkoutRestSec}
	issues := make([]string, 0)

	lines := splitPlanLines(text)
	for i, raw := range lines {
		line := cleanWorkoutLine(raw)
		if line == "" {
			continue
		}
		if i == 0 && !looksLikeWorkoutLine(line) {
			continue
		}
		if isWorkoutHeaderLine(line) {
			continue
		}
		if isRestLine(line) {
			continue
		}
		ex, err := parseWorkoutLine(line)
		if err != "" {
			issues = append(issues, fmt.Sprintf("Строка %d: %s", i+1, err))
			continue
		}
		plan.Exercises = append(plan.Exercises, ex)
	}

	if len(plan.Exercises) == 0 {
		issues = append(issues, "Нет упражнений для тренировки")
	}

	return plan, issues
}

func cleanWorkoutLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	line = regexp.MustCompile(`^\d+[\.)]\s*`).ReplaceAllString(line, "")
	line = regexp.MustCompile(`^[•\-–—]+\s*`).ReplaceAllString(line, "")
	return strings.TrimSpace(line)
}

func looksLikeWorkoutLine(line string) bool {
	if strings.Contains(line, "|") || slashFieldRe.MatchString(line) {
		return true
	}
	if workoutSetsRe.MatchString(line) || workoutDurationRe.MatchString(line) {
		return true
	}
	return false
}

func isRestLine(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	return strings.HasPrefix(lower, "отдых") || strings.HasPrefix(lower, "rest") || strings.HasPrefix(lower, "off")
}

func isWorkoutHeaderLine(line string) bool {
	if workoutHeaderRe.MatchString(line) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(line))
	return strings.HasPrefix(lower, "day ") || strings.HasPrefix(lower, "день ")
}

func parseWorkoutLine(line string) (domain.WorkoutExercise, string) {
	fields := splitWorkoutFields(line)
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	if len(fields) < 2 {
		return domain.WorkoutExercise{}, "используй формат: Название | 3x10 | 60 | 120 (или через /)"
	}
	if len(fields) > 4 {
		return domain.WorkoutExercise{}, "слишком много секций, ожидается максимум 4"
	}

	name := fields[0]
	cardio := false
	if idx := strings.Index(name, ":"); idx != -1 {
		prefix := strings.ToLower(strings.TrimSpace(name[:idx]))
		if prefix == "кардио" || prefix == "cardio" {
			cardio = true
			name = strings.TrimSpace(name[idx+1:])
		}
	}
	if name == "" {
		return domain.WorkoutExercise{}, "пустое название упражнения"
	}
	main := stripNotes(fields[1])
	if main == "" {
		return domain.WorkoutExercise{}, "не указан формат (пример: 3x10 или 25 мин)"
	}

	if cardio {
		dur, ok := parseCardioDurationSec(main)
		if !ok || dur <= 0 {
			return domain.WorkoutExercise{}, "длительность указывается в минутах (например 25 мин)"
		}
		return domain.WorkoutExercise{
			Name:        name,
			Type:        domain.WorkoutExerciseCardio,
			DurationSec: dur,
			RestSec:     defaultWorkoutRestSec,
		}, ""
	}

	if match := workoutSetsRe.FindStringSubmatch(main); len(match) == 3 {
		sets, _ := strconv.Atoi(match[1])
		reps, _ := strconv.Atoi(match[2])
		weight := 0.0
		if len(fields) >= 3 {
			w, ok := parseWeightKg(stripNotes(fields[2]))
			if !ok {
				return domain.WorkoutExercise{}, "вес должен быть числом (кг) или пустым"
			}
			weight = w
		}
		restSec := defaultWorkoutRestSec
		if len(fields) >= 4 {
			if v := strings.TrimSpace(fields[3]); v != "" {
				parsed, ok := parseDurationSec(stripNotes(v), false)
				if !ok {
					return domain.WorkoutExercise{}, "отдых указывается в секундах (например 120) или минутах (например 2 мин)"
				}
				restSec = parsed
			}
		}
		return domain.WorkoutExercise{
			Name:    name,
			Type:    domain.WorkoutExerciseStrength,
			Weight:  weight,
			Reps:    reps,
			Sets:    sets,
			RestSec: restSec,
		}, ""
	}

	if workoutDurationRe.MatchString(main) || looksLikeDurationField(main) {
		dur, ok := parseCardioDurationSec(main)
		if !ok || dur <= 0 {
			return domain.WorkoutExercise{}, "длительность указывается в минутах (например 25 мин)"
		}
		return domain.WorkoutExercise{
			Name:        name,
			Type:        domain.WorkoutExerciseCardio,
			DurationSec: dur,
			RestSec:     defaultWorkoutRestSec,
		}, ""
	}

	return domain.WorkoutExercise{}, "не удалось распознать формат (пример: Название | 3x10 | 60 | 120)"
}

func looksLikeDurationField(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if strings.Contains(lower, "мин") || strings.Contains(lower, "min") || strings.Contains(lower, "сек") || strings.Contains(lower, "sec") {
		return true
	}
	if strings.Contains(lower, ":") {
		return true
	}
	return workoutNumberOnly.MatchString(lower)
}

func parseCardioDurationSec(raw string) (int, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}
	dur, ok := parseDurationSec(trimmed, false)
	if !ok {
		return 0, false
	}
	if workoutNumberOnly.MatchString(trimmed) {
		return dur * 60, true
	}
	return dur, true
}

func splitWorkoutFields(line string) []string {
	if strings.Contains(line, "|") {
		return strings.Split(line, "|")
	}
	if slashFieldRe.MatchString(line) {
		return slashFieldRe.Split(line, -1)
	}
	return []string{line}
}

func stripNotes(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, "("); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}
	return strings.TrimSpace(value)
}

func parseWeightKg(raw string) (float64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "-" || trimmed == "—" {
		return 0, true
	}
	lower := strings.ToLower(trimmed)
	if lower == "bw" || lower == "bodyweight" || lower == "свой вес" {
		return 0, true
	}
	match := workoutNumberRe.FindString(trimmed)
	if match == "" {
		return 0, false
	}
	match = strings.ReplaceAll(match, ",", ".")
	val, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

func parseDurationSec(raw string, requireUnit bool) (int, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}
	if parts := strings.Split(trimmed, ":"); len(parts) == 2 {
		mins, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		secs, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 == nil && err2 == nil {
			return mins*60 + secs, true
		}
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "мин") || strings.Contains(lower, "min") || strings.HasSuffix(lower, "m") {
		match := workoutNumberRe.FindString(lower)
		if match == "" {
			return 0, false
		}
		match = strings.ReplaceAll(match, ",", ".")
		val, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return 0, false
		}
		return int(val * 60), true
	}
	if strings.Contains(lower, "сек") || strings.Contains(lower, "sec") || strings.HasSuffix(lower, "s") {
		match := workoutNumberRe.FindString(lower)
		if match == "" {
			return 0, false
		}
		match = strings.ReplaceAll(match, ",", ".")
		val, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return 0, false
		}
		return int(val), true
	}
	if requireUnit {
		return 0, false
	}
	match := workoutNumberRe.FindString(trimmed)
	if match == "" {
		return 0, false
	}
	match = strings.ReplaceAll(match, ",", ".")
	val, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0, false
	}
	return int(val), true
}
