package handlers

import (
	"barzhafit/internal/service"
	"barzhafit/internal/telegram"
	"barzhafit/internal/util"
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Today struct {
	api     *tgbotapi.BotAPI
	plan    *service.PlanService
	workout *service.WorkoutService
	targets *service.TargetsService
	tz      string
}

func NewToday(api *tgbotapi.BotAPI, plan *service.PlanService, workout *service.WorkoutService, targets *service.TargetsService, tz string) *Today {
	return &Today{api: api, plan: plan, workout: workout, targets: targets, tz: tz}
}

func (h *Today) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID

	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)
	day := util.Weekday1to7(now)

	planText, err := h.plan.Get(ctx, chatID)
	if err != nil {
		h.api.Send(tgbotapi.NewMessage(chatID, "Нет плана. Сначала /plan"))
		return
	}

	days := service.SplitPlanByDays(planText)
	block := strings.TrimSpace(days[day])
	if block == "" {
		block = "День не найден в плане. Проверь, что у тебя строка с числом дня (1..7) стоит отдельно."
	}

	status, has, err := h.workout.GetStatusToday(ctx, chatID, now)
	if err != nil {
		log.Printf("today status failed: chat_id=%d err=%v", chatID, err)
		h.api.Send(tgbotapi.NewMessage(chatID, "Ошибка чтения статуса"))
		return
	}

	st := "—"
	if has {
		if status == "done" {
			st = "✅"
		} else if status == "skip" {
			st = "❌"
		}
	}

	// цели (если нет — покажем дефолт + подсказку)
	kcalTarget := 2400
	proteinTarget := 170
	tg, ok, _ := h.targets.Get(ctx, chatID)
	source := "default"
	if ok {
		kcalTarget = tg.Kcal
		proteinTarget = tg.ProteinG
		source = tg.Source
	}

	text := fmt.Sprintf(
		"День %d\n\n%s\n\nТренировка: %s\n\nКалории: 0 / %d (%s)\nБелок: 0 / %d",
		day, block, st, kcalTarget, source, proteinTarget,
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = telegram.WorkoutButtons()
	h.api.Send(msg)
}
