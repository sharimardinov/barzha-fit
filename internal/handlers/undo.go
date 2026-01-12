package handlers

import (
	"barzhafit/internal/service"
	"context"

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

	ok, err := h.nut.UndoLast(ctx, chatID)
	if err != nil {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Ошибка undo"))
		return
	}
	if !ok {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Нечего удалять"))
		return
	}
	_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Удалил последний приём"))
}
