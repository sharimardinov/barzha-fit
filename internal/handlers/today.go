package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"

	"barzhafit/internal/service"
	"barzhafit/internal/telegram"
	"barzhafit/internal/util"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Today struct {
	api     *tgbotapi.BotAPI
	plan    *service.PlanService
	workout *service.WorkoutService
	tz      string
}

func NewToday(api *tgbotapi.BotAPI, plan *service.PlanService, workout *service.WorkoutService, tz string) *Today {
	return &Today{api: api, plan: plan, workout: workout, tz: tz}
}

func (h *Today) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID

	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)
	day := util.Weekday1to7(now) // Monday=1..Sunday=7

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

	text := fmt.Sprintf("День %d\n\n%s\n\nТренировка: %s", day, block, st)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = telegram.WorkoutButtons()
	h.api.Send(msg)
}
