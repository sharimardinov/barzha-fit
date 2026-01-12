package handlers

import (
	"barzhafit/internal/domain"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type StateSetter interface {
	Set(chatID int64, st domain.State)
}

type Meal struct {
	api   *tgbotapi.BotAPI
	state StateSetter
}

func NewMeal(api *tgbotapi.BotAPI, state StateSetter) *Meal {
	return &Meal{api: api, state: state}
}

func (h *Meal) Handle(m *tgbotapi.Message) {
	chatID := m.Chat.ID

	h.state.Set(chatID, domain.StateWaitMealText)

	_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Напиши одним сообщением что ел."))
}
