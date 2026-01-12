package handlers

import (
	"context"
	"fmt"
	"strings"

	"barzhafit/internal/service"
	"barzhafit/internal/util"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Week struct {
	api  *tgbotapi.BotAPI
	plan *service.PlanService
	tz   string
}

func NewWeek(api *tgbotapi.BotAPI, plan *service.PlanService, tz string) *Week {
	return &Week{api: api, plan: plan, tz: tz}
}

func (h *Week) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID

	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)
	today := util.Weekday1to7(now) // Пн=1..Вс=7

	planText, err := h.plan.Get(ctx, chatID)
	if err != nil {
		h.api.Send(tgbotapi.NewMessage(chatID, "Нет плана. Сначала /plan"))
		return
	}

	days := service.SplitPlanByDays(planText)

	var b strings.Builder
	b.WriteString("Неделя:\n\n")

	for d := 1; d <= 7; d++ {
		prefix := "  "
		if d == today {
			prefix = "👉"
		}
		block := strings.TrimSpace(days[d])
		if block == "" {
			block = "(пусто)"
		}
		// чуть компактнее: отделяем дни линией
		b.WriteString(fmt.Sprintf("%s День %d\n%s\n\n", prefix, d, block))
	}

	msg := tgbotapi.NewMessage(chatID, b.String())
	h.api.Send(msg)
}
