package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Start struct {
	api *tgbotapi.BotAPI
}

func NewStart(api *tgbotapi.BotAPI) *Start {
	return &Start{api: api}
}

func (h *Start) Handle(m *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(m.Chat.ID, "Ок. Команды: /today, /meal, /plan")
	h.api.Send(msg)
}
