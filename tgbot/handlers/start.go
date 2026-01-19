package handlers

import (
	"context"
	"fmt"

	"barzhafit/backend/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Start struct {
	api   *tgbotapi.BotAPI
	users *service.BotUsersService
}

func NewStart(api *tgbotapi.BotAPI, users *service.BotUsersService) *Start {
	return &Start{api: api, users: users}
}

func (h *Start) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	_ = h.users.Ensure(ctx, m.Chat.ID)

	username := h.api.Self.UserName
	link := fmt.Sprintf("https://t.me/%s?startapp", username)
	msg := tgbotapi.NewMessage(m.Chat.ID, "Открыть приложение:\n"+link)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Открыть в полный экран", link),
		),
	)
	_, _ = h.api.Send(msg)
}
