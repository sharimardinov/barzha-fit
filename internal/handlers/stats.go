package handlers

import (
	"context"
	"strings"
	"time"

	"barzhafit/internal/service"
	"barzhafit/internal/telegram"
	"barzhafit/internal/util"

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
	} else if monthArg, ok := parseMonthArg(args); ok {
		monthStart := time.Date(monthArg.Year(), time.Month(monthArg.Month()), 1, 0, 0, 0, 0, loc)
		text, _ = h.view.MonthText(ctx, chatID, now, monthStart, true)
	} else if monthArg, ok := parseMonthArg(strings.TrimPrefix(cmd, "stats")); ok {
		monthStart := time.Date(monthArg.Year(), time.Month(monthArg.Month()), 1, 0, 0, 0, 0, loc)
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

type monthArg struct {
	month int
	year  int
}

func (m monthArg) Month() int { return m.month }
func (m monthArg) Year() int  { return m.year }

func parseMonthArg(s string) (monthArg, bool) {
	if len(s) != 4 {
		return monthArg{}, false
	}
	mm, err := atoi(s[:2])
	if err != nil || mm < 1 || mm > 12 {
		return monthArg{}, false
	}
	yy, err := atoi(s[2:])
	if err != nil {
		return monthArg{}, false
	}
	return monthArg{month: mm, year: 2000 + yy}, true
}

func atoi(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errBadInt
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

var errBadInt = &parseErr{}

type parseErr struct{}

func (e *parseErr) Error() string { return "bad int" }
