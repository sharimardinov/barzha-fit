package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"barzhafit/internal/service"
	"barzhafit/internal/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Today struct {
	api     *tgbotapi.BotAPI
	plan    *service.PlanService
	workout *service.WorkoutService
}

func NewToday(api *tgbotapi.BotAPI, plan *service.PlanService, workout *service.WorkoutService) *Today {
	return &Today{api: api, plan: plan, workout: workout}
}

func (h *Today) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	now := time.Now()

	planText, err := h.plan.Get(ctx, m.Chat.ID)
	if err != nil {
		h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "Нет плана. Сначала /plan"))
		return
	}

	day, status, has, err := h.workout.GetToday(ctx, m.Chat.ID, now)
	if err != nil {
		log.Printf("today get failed: chat_id=%d err=%v", m.Chat.ID, err)
		h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "Ошибка чтения дня"))
		return
	}

	days := service.SplitPlanByDays(planText)
	block := strings.TrimSpace(days[day])
	if block == "" {
		block = "День не найден в плане. Проверь, что есть строка с числом дня (1..7) отдельно."
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

	msg := tgbotapi.NewMessage(m.Chat.ID, text)
	if !has { // если сегодня ещё не отмечал — показываем кнопки
		msg.ReplyMarkup = telegram.WorkoutButtons()
	}
	h.api.Send(msg)
}
