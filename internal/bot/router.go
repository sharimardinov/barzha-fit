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
	if !strings.HasPrefix(m.Text, "/") {
		return false
	}

	cmd := strings.Split(strings.TrimSpace(m.Text), " ")[0]
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
