package handlers

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Help struct {
	api *tgbotapi.BotAPI
}

func NewHelp(api *tgbotapi.BotAPI) *Help {
	return &Help{api: api}
}

func (h *Help) Handle(m *tgbotapi.Message) {
	var b strings.Builder
	b.WriteString("Как считаются цели\n")
	b.WriteString("1) Калории: по профилю и активности (TDEE), затем цель влияет на калории.\n")
	b.WriteString("2) Белок: 2.0 г на кг веса.\n")
	b.WriteString("3) Жир: 1.0 г на кг веса.\n")
	b.WriteString("4) Угли: остаток от калорий после белка и жира.\n")
	b.WriteString("5) Цель: похудение −25%, баланс 0%, набор +20%.\n")
	b.WriteString("6) Шаги: по умолчанию 10 000, можно изменить /targetsset steps <число>.\n\n")
	b.WriteString("Статусы в /status\n")
	b.WriteString("- 🟢 >= 100% цели, 🟡 >= 85%, 🟠 >= 70%, 🔴 ниже 70%.\n")
	b.WriteString("- Дефицит: норм, если >= 85% калорий.\n\n")
	b.WriteString("Команды: см. /start")

	_, _ = h.api.Send(tgbotapi.NewMessage(m.Chat.ID, b.String()))
}
