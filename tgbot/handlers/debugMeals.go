package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"barzhafit/backend/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type DebugMeals struct {
	api *tgbotapi.BotAPI
	nut *service.NutritionService
}

func NewDebugMeals(api *tgbotapi.BotAPI, nut *service.NutritionService) *DebugMeals {
	return &DebugMeals{api: api, nut: nut}
}

func (h *DebugMeals) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID

	items, err := h.nut.ListRecent(ctx, chatID, 10)
	if err != nil {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка: %v", err)))
		return
	}

	timestamps := make([]string, 0, len(items))
	for _, item := range items {
		timestamps = append(timestamps, fmt.Sprintf("Local: %s\nUTC: %s",
			item.EatenAt.Format("2006-01-02 15:04:05 MST"),
			item.EatenAt.UTC().Format("2006-01-02 15:04:05 MST")))
	}

	if len(items) == 0 {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Вообще нет записей в базе"))
		return
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔍 Найдено записей: %d\n\n", len(items)))
	b.WriteString(fmt.Sprintf("Сейчас: %s\n\n", time.Now().Format("2006-01-02 15:04:05 MST")))

	for i, it := range items {
		kcal := calcKcal(it.ProteinG, it.FatG, it.CarbsG)
		b.WriteString(fmt.Sprintf("#%d [ID=%d]\n", i+1, it.ID))
		b.WriteString(fmt.Sprintf("%s\n", timestamps[i]))
		b.WriteString(fmt.Sprintf("%dkcal (Б%d Ж%d У%d)\n", kcal, it.ProteinG, it.FatG, it.CarbsG))
		b.WriteString(fmt.Sprintf("Текст: %s\n\n", it.Text))
	}

	_, _ = h.api.Send(tgbotapi.NewMessage(chatID, b.String()))
}
