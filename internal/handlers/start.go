package handlers

import (
	"context"

	"barzhafit/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Start struct {
	api   *tgbotapi.BotAPI
	users *service.BotUsersService
}

func NewStart(api *tgbotapi.BotAPI, users *service.BotUsersService) *Start {
	return &Start{api: api, users: users}
}

func (h *Start) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	_ = h.users.Ensure(ctx, m.Chat.ID)

	_, _ = h.api.Send(tgbotapi.NewMessage(
		m.Chat.ID,
		"Команды:\n/start\n/help\n/today\n/week\n/meal\n/meals\n/plan\n/planset\n/planshow\n/planday 3\n/profile\n/profileset\n/targets\n/targetsrefresh\n/targetsset kcal 2600\n/undo\n/stats",
	))
}
