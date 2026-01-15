package service

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

var dayHeaderRe = regexp.MustCompile(`(?mi)^\s*(?:[*_#>\-•]+\s*)?(?:день\s*)?([1-7])\s*[:\-–—.]?\s*(?:[*_]+)?\s*$`)

// SplitPlanByDays режет план по строкам, где строка — это "1".."7" или "День 1".."День 7".
// Возвращает map[day]text (текст дня без заголовка).
func SplitPlanByDays(plan string) map[int]string {
	plan = strings.ReplaceAll(plan, "\r\n", "\n")

	if tp, ok := ParseTrainingPlan(plan); ok {
		out := make(map[int]string, 7)
		for i := 0; i < 7; i++ {
			out[i+1] = strings.TrimSpace(tp.Days[i])
		}
		return out
	}

	matches := dayHeaderRe.FindAllStringSubmatchIndex(plan, -1)
	res := make(map[int]string)
	if len(matches) == 0 {
		return res
	}

	for i := 0; i < len(matches); i++ {
		// match: [fullStart fullEnd group1Start group1End]
		g1s, g1e := matches[i][2], matches[i][3]
		dayStr := strings.TrimSpace(plan[g1s:g1e])
		day := int(dayStr[0] - '0')

		start := matches[i][1] // конец строки с номером дня
		end := len(plan)
		if i+1 < len(matches) {
			end = matches[i+1][0] // начало следующего заголовка
		}

		block := strings.TrimSpace(plan[start:end])
		res[day] = block
	}

	return res
}

type TrainingPlan struct {
	Days           []string
	Comment        string
	ExerciseCounts []int
}

type trainingPlanPayload struct {
	Days    []string          `json:"days"`
	Comment string            `json:"comment"`
	Day1    string            `json:"day1"`
	Day2    string            `json:"day2"`
	Day3    string            `json:"day3"`
	Day4    string            `json:"day4"`
	Day5    string            `json:"day5"`
	Day6    string            `json:"day6"`
	Day7    string            `json:"day7"`
	Week    []trainingPlanDay `json:"week_plan"`
}

type trainingPlanDay struct {
	Day        int                 `json:"day"`
	Name       string              `json:"name"`
	Focus      string              `json:"focus"`
	Groups     []trainingPlanGroup `json:"groups"`
	Activities []string            `json:"activities"`
	Notes      string              `json:"notes"`
	Items      []string            `json:"items"`
}

type trainingPlanGroup struct {
	MuscleGroup string                 `json:"muscle_group"`
	Exercises   []trainingPlanExercise `json:"exercises"`
}

type trainingPlanExercise struct {
	Name     string `json:"name"`
	Sets     string `json:"sets"`
	Reps     string `json:"reps"`
	Notes    string `json:"notes"`
	Duration string `json:"duration"`
}

func parseTrainingPlanPayload(plan string) (trainingPlanPayload, bool) {
	raw := strings.TrimSpace(plan)
	if raw == "" || (!strings.HasPrefix(raw, "{") && !strings.HasPrefix(raw, "[")) {
		return trainingPlanPayload{}, false
	}
	var payload trainingPlanPayload
	if err := json.Unmarshal([]byte(sanitizeJSON(raw)), &payload); err != nil {
		return trainingPlanPayload{}, false
	}
	return payload, true
}

func NormalizeTrainingPlan(plan string) (string, bool) {
	payload, ok := parseTrainingPlanPayload(plan)
	if !ok {
		return "", false
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func ParseTrainingPlan(plan string) (TrainingPlan, bool) {
	payload, ok := parseTrainingPlanPayload(plan)
	if !ok {
		return TrainingPlan{}, false
	}

	days := make([]string, 0, 7)
	exCounts := make([]int, 0, 7)
	if len(payload.Week) > 0 {
		for _, day := range payload.Week {
			text, count := formatTrainingDay(day)
			days = append(days, text)
			exCounts = append(exCounts, count)
		}
	} else if len(payload.Days) > 0 {
		days = payload.Days
	} else {
		days = []string{
			payload.Day1,
			payload.Day2,
			payload.Day3,
			payload.Day4,
			payload.Day5,
			payload.Day6,
			payload.Day7,
		}
	}

	if len(days) < 7 {
		for len(days) < 7 {
			days = append(days, "")
		}
	}
	if len(days) > 7 {
		days = days[:7]
	}
	if len(exCounts) > 7 {
		exCounts = exCounts[:7]
	}
	for len(exCounts) < len(days) {
		exCounts = append(exCounts, 0)
	}

	return TrainingPlan{Days: days, Comment: strings.TrimSpace(payload.Comment), ExerciseCounts: exCounts}, true
}

func FormatPlanForDisplay(plan string) (string, bool) {
	tp, ok := ParseTrainingPlan(plan)
	if !ok {
		return "", false
	}
	var b strings.Builder
	for i := 0; i < 7; i++ {
		lines := splitPlanLines(tp.Days[i])
		title := "—"
		body := ""
		if len(lines) > 0 {
			title = strings.TrimSpace(lines[0])
		}
		if len(lines) > 1 {
			body = strings.Join(lines[1:], "\n")
		}
		b.WriteString("ДЕНЬ ")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(" — ")
		b.WriteString(title)
		if body != "" {
			b.WriteString("\n")
			b.WriteString(body)
		}
		if i < 6 {
			b.WriteString("\n\n")
		}
	}
	if tp.Comment != "" {
		// Comment intentionally hidden in UI output.
	}
	return b.String(), true
}

func splitPlanLines(text string) []string {
	raw := strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		clean := strings.TrimSpace(line)
		clean = strings.Trim(clean, "*_")
		if clean == "" {
			continue
		}
		out = append(out, clean)
	}
	return out
}

func sanitizeJSON(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}
	raw = strings.TrimPrefix(raw, "\uFEFF")
	raw = escapeJSONStrings(raw)
	raw = regexp.MustCompile(`,\s*([}\]])`).ReplaceAllString(raw, "$1")
	return raw
}

func escapeJSONStrings(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))
	inString := false
	escape := false
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if escape {
			b.WriteByte(ch)
			escape = false
			continue
		}
		if ch == '\\' && inString {
			escape = true
			b.WriteByte(ch)
			continue
		}
		if ch == '"' {
			inString = !inString
			b.WriteByte(ch)
			continue
		}
		if inString {
			switch ch {
			case '\n':
				b.WriteString(`\n`)
				continue
			case '\r':
				continue
			}
		}
		b.WriteByte(ch)
	}
	return b.String()
}

func formatTrainingDay(day trainingPlanDay) (string, int) {
	var b strings.Builder
	title := strings.TrimSpace(day.Name)
	if title == "" {
		title = "—"
	}
	if strings.TrimSpace(day.Focus) != "" {
		title = title + " (" + strings.TrimSpace(day.Focus) + ")"
	}
	b.WriteString(title)
	b.WriteString("\n")

	counter := 0
	if len(day.Items) > 0 {
		for _, item := range day.Items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			counter++
			b.WriteString(strconv.Itoa(counter))
			b.WriteString(". ")
			b.WriteString(item)
			b.WriteString("\n")
		}
		return strings.TrimSpace(b.String()), counter
	}

	hasExercises := false
	for _, group := range day.Groups {
		groupName := strings.TrimSpace(group.MuscleGroup)
		if groupName != "" {
			b.WriteString(groupName)
			b.WriteString("\n")
		}
		for _, ex := range group.Exercises {
			name := strings.TrimSpace(ex.Name)
			if name == "" {
				continue
			}
			hasExercises = true
			counter++
			b.WriteString(strconv.Itoa(counter))
			b.WriteString(". ")
			b.WriteString(name)
			sets := strings.TrimSpace(ex.Sets)
			reps := strings.TrimSpace(ex.Reps)
			if sets != "" || reps != "" || ex.Duration != "" {
				b.WriteString(" — ")
			}
			if ex.Duration != "" {
				b.WriteString(strings.TrimSpace(ex.Duration))
			} else {
				if sets != "" {
					b.WriteString(sets)
				}
				if reps != "" {
					if sets != "" {
						b.WriteString("x")
					}
					b.WriteString(reps)
				}
			}
			if strings.TrimSpace(ex.Notes) != "" {
				b.WriteString(" (")
				b.WriteString(strings.TrimSpace(ex.Notes))
				b.WriteString(")")
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if !hasExercises && len(day.Activities) > 0 {
		for _, act := range day.Activities {
			act = strings.TrimSpace(act)
			if act == "" {
				continue
			}
			counter++
			b.WriteString(strconv.Itoa(counter))
			b.WriteString(". ")
			b.WriteString(act)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if strings.TrimSpace(day.Notes) != "" {
		b.WriteString(strings.TrimSpace(day.Notes))
	}

	return strings.TrimSpace(b.String()), counter
}
