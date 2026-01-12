package handlers

import (
	"barzhafit/internal/service"
	"barzhafit/internal/telegram"
	"barzhafit/internal/util"
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
	tz      string
}

func NewToday(
	api *tgbotapi.BotAPI,
	plan *service.PlanService,
	workout *service.WorkoutService,
	targets *service.TargetsService,
	nut *service.NutritionService,
	tz string,
) *Today {
	return &Today{
		api:     api,
		plan:    plan,
		workout: workout,
		targets: targets,
		nut:     nut,
		tz:      tz,
	}
}

func (h *Today) Handle(m *tgbotapi.Message) {
	chatID := m.Chat.ID
	ctx := context.Background()

	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)
	dayDate := util.LocalDateStr(now, loc)

	cycleDay, err := h.workout.GetCycleDay(ctx, chatID)
	if err != nil {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Ошибка чтения cycle day"))
		return
	}

	planText, err := h.plan.Get(ctx, chatID)
	if err != nil {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Нет плана. Сначала /plan"))
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

	kcal, p, _, _, err := h.nut.SumToday(ctx, chatID, loc, now)
	if err != nil {
		kcal, p = 0, 0
	}

	kcalTarget := 2400
	proteinTarget := 170
	source := "default"

	tg, ok, _ := h.targets.Get(ctx, chatID)
	if ok {
		kcalTarget = tg.Kcal
		proteinTarget = tg.ProteinG
		source = tg.Source
	}

	text := fmt.Sprintf(
		"День цикла %d\n\n%s\n\nТренировка: %s\n\nКалории: %d / %d (%s)\nБелок: %d / %d",
		cycleDay, block, st, kcal, kcalTarget, source, p, proteinTarget,
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = telegram.WorkoutButtons()
	_, _ = h.api.Send(msg)
}
