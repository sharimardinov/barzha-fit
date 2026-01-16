package handlers

import (
	"context"
	"fmt"
	"strings"

	"barzhafit/backend/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Undo struct {
	api *tgbotapi.BotAPI
	nut *service.NutritionService
}

func NewUndo(api *tgbotapi.BotAPI, nut *service.NutritionService) *Undo {
	return &Undo{api: api, nut: nut}
}

func (h *Undo) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID

	items, err := h.nut.ListRecent(ctx, chatID, 5)
	if err != nil {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Ошибка чтения meals"))
		return
	}
	if len(items) == 0 {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Нечего удалять"))
		return
	}

	var b strings.Builder
	b.WriteString("Выбери приём для удаления:\n")

	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(items))
	for i, it := range items {
		line := fmt.Sprintf("%d) %s", i+1, strings.TrimSpace(it.Text))
		b.WriteString(line + "\n")
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Удалить %d", i+1), fmt.Sprintf("meal:del:%d", it.ID)),
		))
	}

	msg := tgbotapi.NewMessage(chatID, b.String())
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	_, _ = h.api.Send(msg)
}
