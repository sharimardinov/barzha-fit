package handlers

import (
	"barzhafit/internal/service"
	"barzhafit/internal/util"
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Meals struct {
	api *tgbotapi.BotAPI
	nut *service.NutritionService
	tz  string
}

func NewMeals(api *tgbotapi.BotAPI, nut *service.NutritionService, tz string) *Meals {
	return &Meals{api: api, nut: nut, tz: tz}
}

func (h *Meals) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID
	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)

	items, err := h.nut.ListToday(ctx, chatID, loc, now)
	if err != nil {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Ошибка чтения meals"))
		return
	}
	if len(items) == 0 {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "За сегодня пусто"))
		return
	}

	var b strings.Builder
	for _, it := range items {
		b.WriteString(fmt.Sprintf("- %s — %dkcal (Б%d Ж%d У%d)\n",
			strings.TrimSpace(it.Text), it.Kcal, it.ProteinG, it.FatG, it.CarbsG))
	}
	_, _ = h.api.Send(tgbotapi.NewMessage(chatID, b.String()))
}
