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

	// /profile set ...
	if strings.HasPrefix(strings.ToLower(args), "set") {
		text := strings.TrimSpace(strings.TrimPrefix(args, "set"))
		if text == "" {
			h.api.Send(tgbotapi.NewMessage(chatID, "Формат: /profile set рост 180 вес 92 жир 20 возраст 28 активность mid"))
			return
		}

		p, err := h.profile.SaveFromText(ctx, chatID, text)
		if err != nil {
			h.api.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения профиля"))
			return
		}

		// сразу пересчитаем цели
		_, _ = h.targets.Refresh(ctx, chatID)

		h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"Профиль сохранён:\nрост %d, вес %.1f, жир %.1f%%, возраст %d, активность %s",
			p.HeightCM, p.WeightKG, p.BodyFatPct, p.Age, p.Activity,
		)))
		return
	}

	// /profile — показать
	p, ok, err := h.profile.Get(ctx, chatID)
	if err != nil || !ok {
		h.api.Send(tgbotapi.NewMessage(chatID, "Профиля нет. Введи: /profile set рост 180 вес 92 жир 20 возраст 28 активность mid"))
		return
	}

	h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Профиль:\nпол %s, возраст %d\nрост %d\nвес %.1f\nжир %.1f%%\nактивность %s",
		emptyDash(p.Sex), p.Age, p.HeightCM, p.WeightKG, p.BodyFatPct, emptyDash(p.Activity),
	)))
}

func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
