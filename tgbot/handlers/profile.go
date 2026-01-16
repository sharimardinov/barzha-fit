package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"barzhafit/backend/domain"
	"barzhafit/backend/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Profile struct {
	api     *tgbotapi.BotAPI
	state   domain.StateSetter
	drafts  *service.ProfileDraftStore
	profile *service.ProfileService
	targets *service.TargetsService
	plan    *service.PlanService
	ai      *service.ActivityAI
}

func NewProfile(
	api *tgbotapi.BotAPI,
	state domain.StateSetter,
	drafts *service.ProfileDraftStore,
	profile *service.ProfileService,
	targets *service.TargetsService,
	plan *service.PlanService,
	ai *service.ActivityAI,
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
				"Профиль сохранён:\nПол: %s\nВозраст: %d\nРост: %d см\nВес: %.1f кг\nЖир: %.1f%%\nАктивность: %s\nЦель: %s",
				emptyDash(p.Sex), p.Age, p.HeightCM, p.WeightKG, p.BodyFatPct, formatActivity(p.Activity), emptyDash(p.Goal),
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
		h.api.Send(tgbotapi.NewMessage(chatID, "Профиль не найден. Используй /profileset если не лох"))
		return
	}

	h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Твой профиль:\nПол: %s\nВозраст: %d\nРост: %d см\nВес: %.1f кг\nЖир: %.1f%%\nАктивность: %s\nЦель: %s",
		emptyDash(p.Sex), p.Age, p.HeightCM, p.WeightKG, p.BodyFatPct, formatActivity(p.Activity), emptyDash(p.Goal),
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

	draft, _, ok := h.drafts.Snapshot(chatID)
	if !ok {
		_ = h.drafts.SetActivity(chatID, "mid", fmt.Errorf("draft missing"))
		return
	}
	p := domain.Profile{
		ChatID:     chatID,
		Sex:        draft.Sex,
		HeightCM:   draft.HeightCM,
		WeightKG:   draft.WeightKG,
		BodyFatPct: draft.BodyFatPct,
		Age:        draft.Age,
	}

	mult, _, err := h.ai.EstimateActivityMultiplierWithProfile(ctx, planText, p)
	if err != nil {
		log.Printf("activity estimate failed: chatID=%d err=%v", chatID, err)
		_ = h.drafts.SetActivity(chatID, "mid", err)
		return
	}

	activity := fmt.Sprintf("ai:%.2f", mult)
	_ = h.drafts.SetActivity(chatID, activity, nil)
}
