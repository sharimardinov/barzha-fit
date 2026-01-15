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
	Days    []string
	Comment string
}

type trainingPlanPayload struct {
	Days    []string `json:"days"`
	Comment string   `json:"comment"`
	Day1    string   `json:"day1"`
	Day2    string   `json:"day2"`
	Day3    string   `json:"day3"`
	Day4    string   `json:"day4"`
	Day5    string   `json:"day5"`
	Day6    string   `json:"day6"`
	Day7    string   `json:"day7"`
}

func ParseTrainingPlan(plan string) (TrainingPlan, bool) {
	raw := strings.TrimSpace(plan)
	if raw == "" || (!strings.HasPrefix(raw, "{") && !strings.HasPrefix(raw, "[")) {
		return TrainingPlan{}, false
	}

	var payload trainingPlanPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return TrainingPlan{}, false
	}

	days := make([]string, 0, 7)
	if len(payload.Days) > 0 {
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

	return TrainingPlan{Days: days, Comment: strings.TrimSpace(payload.Comment)}, true
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
		b.WriteString("\n\nКомментарий:\n")
		b.WriteString(tp.Comment)
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
