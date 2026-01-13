package handlers

import (
	"context"
	"strings"

	"barzhafit/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Hard struct {
	api   *tgbotapi.BotAPI
	users *service.BotUsersService
}

func NewHard(api *tgbotapi.BotAPI, users *service.BotUsersService) *Hard {
	return &Hard{api: api, users: users}
}

func (h *Hard) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID

	_ = h.users.Ensure(ctx, chatID)

	args := strings.TrimSpace(m.CommandArguments())
	args = strings.ToLower(args)

	switch args {
	case "on":
		_ = h.users.SetHard(ctx, chatID, true)
		h.api.Send(tgbotapi.NewMessage(chatID, "Жёсткий режим: ON"))
	case "off":
		_ = h.users.SetHard(ctx, chatID, false)
		h.api.Send(tgbotapi.NewMessage(chatID, "Жёсткий режим: OFF"))
	default:
		h.api.Send(tgbotapi.NewMessage(chatID, "Используй: /hard on или /hard off"))
	}
}
