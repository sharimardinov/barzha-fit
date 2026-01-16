package handlers

import (
	"context"
	"strings"
	"time"

	"barzhafit/backend/input"
	"barzhafit/backend/service"
	"barzhafit/backend/util"
	"barzhafit/tgbot/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Stats struct {
	api  *tgbotapi.BotAPI
	view *service.StatsViewService
	tz   string
}

func NewStats(api *tgbotapi.BotAPI, view *service.StatsViewService, tz string) *Stats {
	return &Stats{api: api, view: view, tz: tz}
}

func (h *Stats) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID
	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)
	cmd := strings.ToLower(strings.TrimSpace(m.Command()))
	args := strings.ToLower(strings.TrimSpace(m.CommandArguments()))
	cmdStats := strings.TrimPrefix(cmd, "stats")
	if strings.HasPrefix(cmdStats, "_") {
		cmdStats = strings.TrimPrefix(cmdStats, "_")
	}

	text := ""
	if args == "prevweek" || cmdStats == "prevweek" {
		text, _ = h.view.WeekText(ctx, chatID, now, true)
	} else if args == "prevmonth" || cmdStats == "prevmonth" {
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, -1, 0)
		text, _ = h.view.MonthText(ctx, chatID, now, monthStart, true)
	} else if month, year, ok := input.ParseMonthArg(args); ok {
		monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
		text, _ = h.view.MonthText(ctx, chatID, now, monthStart, true)
	} else if month, year, ok := input.ParseMonthArg(strings.TrimPrefix(cmd, "stats")); ok {
		monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
		text, _ = h.view.MonthText(ctx, chatID, now, monthStart, true)
	} else {
		text, _ = h.view.WeekText(ctx, chatID, now, false)
	}

	msg := tgbotapi.NewMessage(chatID, "📊 Статистика\n\n"+text)
	msg.ParseMode = tgbotapi.ModeHTML
	kb := telegram.StatsButtons()
	msg.ReplyMarkup = kb
	_, _ = h.api.Send(msg)
}
