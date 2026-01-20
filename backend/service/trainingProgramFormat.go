package service

import (
	"fmt"
	"strings"

	"barzhafit/backend/domain"
)

func FormatGeneratedProgram(program domain.GeneratedProgram) string {
	var b strings.Builder
	for i, day := range program.Days {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%d\n", day.Day))
		label := strings.TrimSpace(day.Name)
		if label == "" || strings.HasPrefix(strings.ToLower(label), "day ") {
			label = strings.TrimSpace(day.Focus)
		}
		if label == "" {
			label = "Training"
		}
		b.WriteString(label)
		b.WriteString("\n")
		for j, ex := range day.Exercises {
			line := fmt.Sprintf(
				"%d. %s — %dx%s | RPE %s | Rest %s",
				j+1,
				ex.Name,
				ex.Sets,
				ex.Reps,
				ex.RPE,
				ex.Rest,
			)
			if ex.Percent1RM != "" {
				line += fmt.Sprintf(" | %%1RM %s", ex.Percent1RM)
			}
			if len(ex.Tags) > 0 {
				line += " | " + formatTags(ex.Tags)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return strings.TrimSpace(b.String())
}

func formatTags(tags []string) string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		t := strings.TrimSpace(tag)
		if t == "" {
			continue
		}
		if !strings.HasPrefix(t, "#") {
			t = "#" + t
		}
		out = append(out, t)
	}
	return strings.Join(out, " ")
}
