package handlers

import (
	"barzhafit/backend/service"
	"barzhafit/backend/util"
	"barzhafit/bot/telegram"
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Today struct {
	api     *tgbotapi.BotAPI
	plan    *service.PlanService
	workout *service.WorkoutService
	targets *service.TargetsService
	nut     *service.NutritionService
	steps   *service.StepsService
	tz      string
}

func NewToday(
	api *tgbotapi.BotAPI,
	plan *service.PlanService,
	workout *service.WorkoutService,
	targets *service.TargetsService,
	nut *service.NutritionService,
	steps *service.StepsService,
	tz string,
) *Today {
	return &Today{
		api:     api,
		plan:    plan,
		workout: workout,
		targets: targets,
		nut:     nut,
		steps:   steps,
		tz:      tz,
	}
}

func (h *Today) Handle(m *tgbotapi.Message) {
	chatID := m.Chat.ID
	ctx := context.Background()

	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)
	dayDate := util.LocalDateStr(now, loc)
	cycleDay := util.Weekday1to7(now)

	planText, err := h.plan.Get(ctx, chatID)
	if err != nil {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Нет плана. Сначала /setplan"))
		return
	}

	days := service.SplitPlanByDays(planText)
	block := strings.TrimSpace(days[cycleDay])
	if block == "" {
		block = "День не найден в плане. Проверь заголовки 1..7 отдельной строкой."
	}

	status, has, _ := h.workout.GetStatusByDate(ctx, chatID, dayDate)
	st := "—"
	if has {
		if status == "done" {
			st = "✅"
		} else if status == "skip" {
			st = "❌"
		}
	}

	kcal, p, f, c, err := h.nut.SumToday(ctx, chatID, loc, now)
	if err != nil {
		kcal, p, f, c = 0, 0, 0, 0
	}
	steps, hasSteps, _ := h.steps.GetByDate(ctx, chatID, dayDate)
	if !hasSteps {
		steps = 0
	}

	kcalTarget := 2400
	proteinTarget := 170
	fatTarget := 70
	carbsTarget := 250
	stepsTarget := 10000

	tg, ok, _ := h.targets.Get(ctx, chatID)
	if ok {
		kcalTarget = tg.Kcal
		proteinTarget = tg.ProteinG
		fatTarget = tg.FatG
		carbsTarget = tg.CarbsG
		if tg.Steps > 0 {
			stepsTarget = tg.Steps
		}
	}

	kcalIcon := ratioIcon(float64(kcal), float64(kcalTarget))
	proteinIcon := proteinRatioIcon(float64(p), float64(proteinTarget))
	fatIcon := ratioIcon(float64(f), float64(fatTarget))
	carbsIcon := ratioIcon(float64(c), float64(carbsTarget))
	stepsIcon := ratioIcon(float64(steps), float64(stepsTarget))

	var b strings.Builder
	b.WriteString(fmt.Sprintf("День цикла %d\n\n%s\n\n", cycleDay, block))
	b.WriteString("Сегодня:\n")
	b.WriteString(fmt.Sprintf("Тренировка: %s\n", st))
	b.WriteString(fmt.Sprintf("Калории: %d / %d %s\n", kcal, kcalTarget, kcalIcon))
	b.WriteString(fmt.Sprintf("Белок: %d / %d %s\n", p, proteinTarget, proteinIcon))
	b.WriteString(fmt.Sprintf("Жир: %d / %d %s\n", f, fatTarget, fatIcon))
	b.WriteString(fmt.Sprintf("Углеводы: %d / %d %s\n", c, carbsTarget, carbsIcon))
	b.WriteString(fmt.Sprintf("Шаги: %s / %s %s\n", formatInt(steps), formatInt(stepsTarget), stepsIcon))
	b.WriteString(fmt.Sprintf("Еда: %s", kcalIcon))

	msg := tgbotapi.NewMessage(chatID, b.String())
	msg.ReplyMarkup = telegram.WorkoutButtons()
	_, _ = h.api.Send(msg)
}

func ratioIcon(val, target float64) string {
	if target <= 0 {
		return "—"
	}
	r := val / target
	if r >= 0.9 && r <= 1.1 {
		return "🟢"
	}
	return "🔴"
}

func proteinRatioIcon(val, target float64) string {
	if target <= 0 {
		return "—"
	}
	r := val / target
	if r >= 0.9 {
		return "🟢"
	}
	return "🔴"
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
