package handlers

import (
	"barzhafit/internal/service"
	"barzhafit/internal/util"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

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

	// ДЕБАГ: выведем временные рамки
	from := util.DayStart(now, loc)
	to := from.Add(24 * time.Hour)
	log.Printf("DEBUG /meals: chatID=%d now=%s from=%s to=%s",
		chatID, now.Format("2006-01-02 15:04:05 MST"),
		from.Format("2006-01-02 15:04:05 MST"),
		to.Format("2006-01-02 15:04:05 MST"))

	items, err := h.nut.ListToday(ctx, chatID, loc, now)
	if err != nil {
		log.Printf("ERROR /meals: chatID=%d err=%v", chatID, err)
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Ошибка чтения meals"))
		return
	}

	log.Printf("DEBUG /meals: chatID=%d found %d items", chatID, len(items))

	if len(items) == 0 {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Сегодня ты голодаешь"))
		return
	}

	var b strings.Builder
	for _, it := range items {
		b.WriteString(fmt.Sprintf("- %s — %dkcal (Б%d Ж%d У%d)\n",
			strings.TrimSpace(it.Text), it.Kcal, it.ProteinG, it.FatG, it.CarbsG))
	}

	kcal, p, f, c, err := h.nut.SumToday(ctx, chatID, loc, now)
	if err != nil {
		log.Printf("ERROR /meals sum: chatID=%d err=%v", chatID, err)
	} else {
		b.WriteString(fmt.Sprintf("\nИтого за сегодня: %dkcal (Б%d Ж%d У%d)",
			kcal, p, f, c))
	}
	_, _ = h.api.Send(tgbotapi.NewMessage(chatID, b.String()))
}
