package handlers

import (
	"barzhafit/internal/domain"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Meal struct {
	api   *tgbotapi.BotAPI
	state domain.StateSetter
}

func NewMeal(api *tgbotapi.BotAPI, state domain.StateSetter) *Meal {
	return &Meal{api: api, state: state}
}

func (h *Meal) Handle(m *tgbotapi.Message) {
	h.state.Set(m.Chat.ID, domain.StateWaitMealText)
	msg := tgbotapi.NewMessage(m.Chat.ID, "Напиши одним сообщением что ел.")
	h.api.Send(msg)
}
