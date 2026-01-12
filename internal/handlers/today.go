package handlers

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Today struct {
	api *tgbotapi.BotAPI
}

func NewToday(api *tgbotapi.BotAPI) *Today {
	return &Today{api: api}
}

func (h *Today) Handle(m *tgbotapi.Message) {
	// пока без БД — заглушка
	now := time.Now().Format("2006-01-02 15:04")
	text := "Сегодня:\nТренировка: —\nКалории: 0 / 2400 🟡\nБелок: 0 / 170 🟡\n\n" + now
	msg := tgbotapi.NewMessage(m.Chat.ID, text)
	h.api.Send(msg)
}
