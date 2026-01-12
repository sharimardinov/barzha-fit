package handlers

import (
	"context"
	"strings"

	"barzhafit/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Morning struct {
	api   *tgbotapi.BotAPI
	users *service.BotUsersService
}

func NewMorning(api *tgbotapi.BotAPI, users *service.BotUsersService) *Morning {
	return &Morning{api: api, users: users}
}

func (h *Morning) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID

	// на всякий: чтобы пользователь точно был в таблице
	_ = h.users.Ensure(ctx, chatID)

	args := strings.TrimSpace(m.CommandArguments())
	args = strings.ToLower(args)

	switch args {
	case "on":
		_ = h.users.SetMorning(ctx, chatID, true)
		h.api.Send(tgbotapi.NewMessage(chatID, "Ок. Утренние сообщения: ON"))
	case "off":
		_ = h.users.SetMorning(ctx, chatID, false)
		h.api.Send(tgbotapi.NewMessage(chatID, "Ок. Утренние сообщения: OFF"))
	default:
		h.api.Send(tgbotapi.NewMessage(chatID, "Используй: /morning on или /morning off"))
	}
}
