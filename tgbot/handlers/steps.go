package handlers

import (
	"barzhafit/backend/domain"
	"barzhafit/backend/service"
	"barzhafit/backend/util"
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Steps struct {
	api   *tgbotapi.BotAPI
	state domain.StateSetter
	steps *service.StepsService
	tz    string
}

func NewSteps(api *tgbotapi.BotAPI, state domain.StateSetter, steps *service.StepsService, tz string) *Steps {
	return &Steps{api: api, state: state, steps: steps, tz: tz}
}

func (h *Steps) Handle(m *tgbotapi.Message) {
	chatID := m.Chat.ID
	args := strings.TrimSpace(m.CommandArguments())
	if args == "" {
		h.state.Set(chatID, domain.StateWaitStepsCount)
		h.api.Send(tgbotapi.NewMessage(chatID, "Сколько нашагал сегодня? Ответь числом например 8500"))
		return
	}

	steps, ok := parseSteps(args)
	if !ok {
		h.api.Send(tgbotapi.NewMessage(chatID, "Напиши количество шагов числом, например 8500"))
		return
	}

	h.state.Clear(chatID)
	loc := util.MustLocation(h.tz)
	dayDate := util.LocalDateStr(util.NowIn(loc), loc)
	if err := h.steps.SetSteps(context.Background(), chatID, dayDate, steps); err != nil {
		h.api.Send(tgbotapi.NewMessage(chatID, "Не удалось сохранить твои шаги за гиги."))
		return
	}

	h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Ок, записал: %d шагов", steps)))
}

func parseSteps(s string) (int, bool) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v < 0 || v > 100000 {
		return 0, false
	}
	return v, true
}
