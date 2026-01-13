package handlers

import (
	"barzhafit/internal/service"
	"barzhafit/internal/util"
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Status struct {
	api     *tgbotapi.BotAPI
	workout *service.WorkoutService
	targets *service.TargetsService
	nut     *service.NutritionService
	steps   *service.StepsService
	tz      string
}

func NewStatus(api *tgbotapi.BotAPI, workout *service.WorkoutService, targets *service.TargetsService, nut *service.NutritionService, steps *service.StepsService, tz string) *Status {
	return &Status{api: api, workout: workout, targets: targets, nut: nut, steps: steps, tz: tz}
}

func (h *Status) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID
	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)
	dayDate := util.LocalDateStr(now, loc)

	status, hasWorkout, _ := h.workout.GetStatusByDate(ctx, chatID, dayDate)
	workoutIcon := "—"
	if hasWorkout {
		if status == "done" {
			workoutIcon = "✅"
		} else if status == "skip" {
			workoutIcon = "❌"
		}
	}

	kcal, p, _, _, _ := h.nut.SumToday(ctx, chatID, loc, now)
	steps, hasSteps, _ := h.steps.GetByDate(ctx, chatID, dayDate)
	if !hasSteps {
		steps = 0
	}

	kcalTarget := 2400
	proteinTarget := 170
	stepsTarget := 10000
	if tg, ok, _ := h.targets.Get(ctx, chatID); ok {
		kcalTarget = tg.Kcal
		proteinTarget = tg.ProteinG
		if tg.Steps > 0 {
			stepsTarget = tg.Steps
		}
	}

	kcalIcon := ratioIcon(float64(kcal), float64(kcalTarget))
	proteinIcon := ratioIcon(float64(p), float64(proteinTarget))
	stepsIcon := ratioIcon(float64(steps), float64(stepsTarget))

	deficit := deficitText(kcal, kcalTarget)

	var b strings.Builder
	b.WriteString("Сегодня:\n")
	b.WriteString(fmt.Sprintf("Тренировка: %s\n", workoutIcon))
	b.WriteString(fmt.Sprintf("Калории: %d / %d %s\n", kcal, kcalTarget, kcalIcon))
	b.WriteString(fmt.Sprintf("Белок: %d / %d %s\n", p, proteinTarget, proteinIcon))
	b.WriteString(fmt.Sprintf("Шаги: %s / %s %s\n", formatInt(steps), formatInt(stepsTarget), stepsIcon))
	b.WriteString(fmt.Sprintf("Дефицит: %s", deficit))

	_, _ = h.api.Send(tgbotapi.NewMessage(chatID, b.String()))
}

func ratioIcon(val, target float64) string {
	if target <= 0 {
		return "—"
	}
	r := val / target
	switch {
	case r >= 1.0:
		return "🟢"
	case r >= 0.85:
		return "🟡"
	case r >= 0.7:
		return "🟠"
	default:
		return "🔴"
	}
}

func deficitText(kcal, target int) string {
	if target <= 0 {
		return "—"
	}
	if kcal > target {
		return "перебор"
	}
	if float64(kcal) >= 0.85*float64(target) {
		return "норм"
	}
	return "дефицит"
}

func formatInt(v int) string {
	s := fmt.Sprintf("%d", v)
	if v < 1000 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, " ")
}
