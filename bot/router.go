package bot

import (
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerFunc func(m *tgbotapi.Message)

type Router struct {
	handlers map[string]HandlerFunc
}

func NewRouter() *Router {
	return &Router{handlers: make(map[string]HandlerFunc)}
}

func (r *Router) Handle(cmd string, h HandlerFunc) {
	r.handlers[cmd] = h
}

func (r *Router) Dispatch(m *tgbotapi.Message) bool {
	if m == nil || m.Text == "" {
		return false
	}
	text := strings.TrimSpace(m.Text)
	if !strings.HasPrefix(text, "/") {
		switch text {
		case "📅 План на сегодня":
			text = "/today"
		case "🍽 Добавить еду":
			text = "/setmeal"
		case "👟 Добавить шаги":
			text = "/setstep"
		case "📊 Статистика недели":
			text = "/stats"
		default:
			return false
		}
	}

	cmd := strings.Split(text, " ")[0]
	log.Printf("CMD chatID=%d cmd=%s args=%q", m.Chat.ID, cmd, strings.TrimSpace(m.CommandArguments()))
	h, ok := r.handlers[cmd]
	if !ok {
		if strings.HasPrefix(cmd, "/stats") {
			if h, ok := r.handlers["/stats"]; ok {
				h(m)
				return true
			}
		}
		return false
	}
	h(m)
	return true
}
