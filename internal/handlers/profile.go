package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"barzhafit/internal/domain"
	"barzhafit/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Profile struct {
	api     *tgbotapi.BotAPI
	state   domain.StateSetter
	drafts  *service.ProfileDraftStore
	profile *service.ProfileService
	targets *service.TargetsService
	plan    *service.PlanService
	ai      *service.AIService
}

func NewProfile(
	api *tgbotapi.BotAPI,
	state domain.StateSetter,
	drafts *service.ProfileDraftStore,
	profile *service.ProfileService,
	targets *service.TargetsService,
	plan *service.PlanService,
	ai *service.AIService,
) *Profile {
	return &Profile{
		api:     api,
		state:   state,
		drafts:  drafts,
		profile: profile,
		targets: targets,
		plan:    plan,
		ai:      ai,
	}
}

func (h *Profile) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID
	cmd := strings.ToLower(m.Command())

	args := strings.TrimSpace(m.CommandArguments())

	// /profileset [<данные>]
	if cmd == "profileset" || strings.HasPrefix(args, "set") {
		text := strings.TrimSpace(strings.TrimPrefix(args, "set"))
		if text != "" {
			p, err := h.profile.SaveFromText(ctx, chatID, text)
			if err != nil {
				h.api.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения"))
				return
			}

			h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf(
				"Профиль сохранён:\nПол: %s\nВозраст: %d\nРост: %d см\nВес: %.1f кг\nЖир: %.1f%%\nАктивность: %s",
				emptyDash(p.Sex), p.Age, p.HeightCM, p.WeightKG, p.BodyFatPct, formatActivity(p.Activity),
			)))
			return
		}

		h.drafts.Start(chatID)
		h.state.Set(chatID, domain.StateWaitProfileSex)
		h.api.Send(tgbotapi.NewMessage(chatID, "Давай настроим профиль. Пол (м/ж)?"))

		go h.prefetchActivity(chatID)
		return
	}

	// /profile — показать
	p, ok, err := h.profile.Get(ctx, chatID)
	if err != nil || !ok {
		h.api.Send(tgbotapi.NewMessage(chatID, "Профиль не найден. Используй /profileset ..."))
		return
	}

	h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Твой профиль:\nПол: %s\nВозраст: %d\nРост: %d см\nВес: %.1f кг\nЖир: %.1f%%\nАктивность: %s",
		emptyDash(p.Sex), p.Age, p.HeightCM, p.WeightKG, p.BodyFatPct, formatActivity(p.Activity),
	)))
}

func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func formatActivity(a string) string {
	if strings.HasPrefix(strings.ToLower(a), "ai:") {
		return strings.TrimPrefix(a, "ai:") + " (ai)"
	}
	return a
}

func (h *Profile) prefetchActivity(chatID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	planText, err := h.plan.Get(ctx, chatID)
	if err != nil || strings.TrimSpace(planText) == "" {
		if err != nil {
			log.Printf("activity estimate: no plan chatID=%d err=%v", chatID, err)
		}
		_ = h.drafts.SetActivity(chatID, "mid", err)
		return
	}

	mult, _, err := h.ai.EstimateActivityMultiplier(ctx, planText)
	if err != nil {
		log.Printf("activity estimate failed: chatID=%d err=%v", chatID, err)
		_ = h.drafts.SetActivity(chatID, "mid", err)
		return
	}

	activity := fmt.Sprintf("ai:%.2f", mult)
	_ = h.drafts.SetActivity(chatID, activity, nil)
}
