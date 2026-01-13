package handlers

import (
	"context"
	"strings"

	"barzhafit/internal/domain"
	"barzhafit/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Plan struct {
	api   *tgbotapi.BotAPI
	state domain.StateSetter
	plan  *service.PlanService
}

func NewPlan(api *tgbotapi.BotAPI, state domain.StateSetter, plan *service.PlanService) *Plan {
	return &Plan{api: api, state: state, plan: plan}
}

func (h *Plan) Handle(m *tgbotapi.Message) {
	cmd := strings.ToLower(m.Command())
	args := strings.TrimSpace(m.CommandArguments())

	if cmd == "planday" || strings.HasPrefix(args, "day") {
		parts := strings.Fields(args) // ["day", "3"] or ["3"]
		if len(parts) == 1 && cmd == "planday" {
			parts = []string{"day", parts[0]}
		}
		if len(parts) != 2 {
			h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "Используй: /planday 3"))
			return
		}

		n := parts[1]
		if len(n) != 1 || n[0] < '1' || n[0] > '7' {
			h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "День должен быть 1..7"))
			return
		}
		day := int(n[0] - '0')

		planText, err := h.plan.Get(context.Background(), m.Chat.ID)
		if err != nil {
			h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "План не найден. Сделай /plan и вставь текст."))
			return
		}

		days := service.SplitPlanByDays(planText)
		block, ok := days[day]
		if !ok || strings.TrimSpace(block) == "" {
			h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "День не найден в плане. Проверь, что у тебя есть строка с числом "+n+" отдельно."))
			return
		}

		h.api.Send(tgbotapi.NewMessage(m.Chat.ID, block))
		return
	}

	// /planshow
	if cmd == "planshow" || args == "show" {
		text, err := h.plan.Get(context.Background(), m.Chat.ID)
		if err != nil {
			h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "План не найден. Сделай /plan и вставь текст."))
			return
		}
		h.api.Send(tgbotapi.NewMessage(m.Chat.ID, text))
		return
	}

	// /plan или /planset -> ждём текст плана
	if cmd == "planset" || args == "" || args == "set" {
		h.state.Set(m.Chat.ID, domain.StateWaitPlanText)
		h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "Вставь план одним сообщением. Дни 1..7, формат свободный."))
		return
	}

	h.api.Send(tgbotapi.NewMessage(m.Chat.ID, "Команды:\n/plan — вставить план\n/planset — вставить план\n/planshow — показать план\n/planday 3 — показать день"))
}
