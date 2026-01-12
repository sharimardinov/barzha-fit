package handlers

import (
	"context"

	"barzhafit/internal/service"

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

	_, _ = h.api.Send(tgbotapi.NewMessage(
		m.Chat.ID,
		"Ок. Команды: /today, /week, /meal, /plan",
	))
}
