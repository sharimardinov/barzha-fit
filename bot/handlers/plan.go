package handlers

import (
	"context"
	"strings"

	"barzhafit/backend/domain"
	"barzhafit/backend/service"
	"barzhafit/backend/util"
	"barzhafit/bot/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Plan struct {
	api   *tgbotapi.BotAPI
	state domain.StateSetter
	plan  *service.PlanService
	tz    string
}

func NewPlan(api *tgbotapi.BotAPI, state domain.StateSetter, plan *service.PlanService, tz string) *Plan {
	return &Plan{api: api, state: state, plan: plan, tz: tz}
}

func (h *Plan) Handle(m *tgbotapi.Message) {
	cmd := strings.ToLower(m.Command())
	args := strings.TrimSpace(m.CommandArguments())

	if cmd == "setplan" {
		h.state.Set(m.Chat.ID, domain.StateWaitPlanText)
		h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "Вставь план одним сообщением. Дни 1..7, формат свободный типа как:\n1\nСкручивания 20/20/20\nБоковая гиперэкстернзия 15/15/15\n\n2\nПодтягивания -20кг 12/12/12\n..."))
		return
	}

	if cmd != "plan" || args != "" {
		h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "Вставь план одним сообщением. Дни 1..7, формат свободный типа как:\n1\nСкручивания 20/20/20\nБоковая гиперэкстернзия 15/15/15\n\n2\nПодтягивания -20кг 12/12/12\n..."))
		return
	}

	ctx := context.Background()
	chatID := m.Chat.ID

	planText, err := h.plan.Get(ctx, chatID)
	if err != nil {
		h.api.Send(tgbotapi.NewMessage(chatID, "Нет плана. Сначала /setplan"))
		return
	}

	if formatted, ok := service.FormatPlanForDisplay(planText); ok {
		planText = formatted
	}
	msg := tgbotapi.NewMessage(chatID, planText)
	loc := util.MustLocation(h.tz)
	day := util.Weekday1to7(util.NowIn(loc))
	msg.ReplyMarkup = telegram.PlanNavButtons(day)
	h.api.Send(msg)
}
