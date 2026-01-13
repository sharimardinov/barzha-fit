package handlers

import (
	"context"
	"strings"

	"barzhafit/internal/domain"
	"barzhafit/internal/service"
	"barzhafit/internal/telegram"
	"barzhafit/internal/util"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Plan struct {
	api   *tgbotapi.BotAPI
	state domain.StateSetter
	view  *service.PlanViewService
	tz    string
}

func NewPlan(api *tgbotapi.BotAPI, state domain.StateSetter, view *service.PlanViewService, tz string) *Plan {
	return &Plan{api: api, state: state, view: view, tz: tz}
}

func (h *Plan) Handle(m *tgbotapi.Message) {
	cmd := strings.ToLower(m.Command())
	args := strings.TrimSpace(m.CommandArguments())

	if cmd == "setplan" {
		h.state.Set(m.Chat.ID, domain.StateWaitPlanText)
		h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "Вставь план одним сообщением. Дни 1..7, формат свободный."))
		return
	}

	if cmd != "plan" || args != "" {
		h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "Команды:\n/plan — показать план\n/setplan — вставить план"))
		return
	}

	ctx := context.Background()
	chatID := m.Chat.ID
	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)
	day := util.Weekday1to7(now)
	text, err := h.view.DayText(ctx, chatID, day, now)
	if err != nil {
		h.api.Send(tgbotapi.NewMessage(chatID, "Нет плана. Сначала /setplan"))
		return
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = telegram.PlanNavButtons(day)
	h.api.Send(msg)
}
