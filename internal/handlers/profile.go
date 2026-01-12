package handlers

import (
	"context"
	"fmt"
	"strings"

	"barzhafit/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Profile struct {
	api     *tgbotapi.BotAPI
	profile *service.ProfileService
	targets *service.TargetsService
}

func NewProfile(api *tgbotapi.BotAPI, profile *service.ProfileService, targets *service.TargetsService) *Profile {
	return &Profile{api: api, profile: profile, targets: targets}
}

func (h *Profile) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID

	args := strings.TrimSpace(m.CommandArguments())

	// /profile set <данные>
	if strings.HasPrefix(args, "set ") {
		text := strings.TrimPrefix(args, "set ")
		text = strings.TrimSpace(text)
		if text == "" {
			h.api.Send(tgbotapi.NewMessage(chatID, "Формат: /profile set пол:м возраст:30 рост:180 вес:85 жир:15 активность:mid"))
			return
		}

		p, err := h.profile.SaveFromText(ctx, chatID, text)
		if err != nil {
			h.api.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения"))
			return
		}

		h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"Профиль сохранён:\nПол: %s\nВозраст: %d\nРост: %d см\nВес: %.1f кг\nЖир: %.1f%%\nАктивность: %s",
			emptyDash(p.Sex), p.Age, p.HeightCM, p.WeightKG, p.BodyFatPct, p.Activity,
		)))
		return
	}

	// /profile — показать
	p, ok, err := h.profile.Get(ctx, chatID)
	if err != nil || !ok {
		h.api.Send(tgbotapi.NewMessage(chatID, "Профиль не найден. Используй /profile set ..."))
		return
	}

	h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Твой профиль:\nПол: %s\nВозраст: %d\nРост: %d см\nВес: %.1f кг\nЖир: %.1f%%\nАктивность: %s",
		emptyDash(p.Sex), p.Age, p.HeightCM, p.WeightKG, p.BodyFatPct, p.Activity,
	)))
}

func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
