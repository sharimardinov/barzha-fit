package handlers

import (
	"context"
	"fmt"
	"strings"

	"barzhafit/backend/domain"
	"barzhafit/backend/input"
	"barzhafit/backend/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Weight struct {
	api     *tgbotapi.BotAPI
	state   domain.StateSetter
	profile *service.ProfileService
}

func NewWeight(api *tgbotapi.BotAPI, state domain.StateSetter, profile *service.ProfileService) *Weight {
	return &Weight{api: api, state: state, profile: profile}
}

func (h *Weight) Handle(m *tgbotapi.Message) {
	chatID := m.Chat.ID
	args := strings.TrimSpace(m.CommandArguments())
	if args == "" {
		h.state.Set(chatID, domain.StateWaitWeightUpdate)
		h.api.Send(tgbotapi.NewMessage(chatID, "Введи вес в кг, например 82.5"))
		return
	}

	weight, ok := input.ParseWeight(args)
	if !ok {
		h.api.Send(tgbotapi.NewMessage(chatID, "Введи вес в кг, например 82.5"))
		return
	}

	h.state.Clear(chatID)
	p, ok, err := h.profile.UpdateWeight(context.Background(), chatID, weight)
	if err != nil {
		h.api.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения"))
		return
	}
	if !ok {
		h.api.Send(tgbotapi.NewMessage(chatID, "Сначала /profileset"))
		return
	}

	h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Вес обновлён: %.1f кг. Если нужно — /targetsrefresh", p.WeightKG)))
}
