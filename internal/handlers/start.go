package handlers

import (
	"context"

	"barzhafit/internal/service"
	"barzhafit/internal/telegram"

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

	msg := tgbotapi.NewMessage(
		m.Chat.ID,
		"Команды\n\nДень\n/today — тренировка + прогресс\n/setstep — шаги за сегодня\n/setmeal — добавить еду\n/meals — еда за сегодня\n/undo — удалить последний прием\n\nПлан\n/plan — показать план\n/setplan — вставить план\n\nПрофиль\n/profile — профиль\n/profileset — заполнить профиль\n/weight — обновить вес\n/targets — цели\n/targetsrefresh — пересчитать цели\n/targetsset kcal <число> — задать калории вручную\n/targetsset steps <число> — задать шаги вручную\n\nИстория\n/streak — серии\n\nСтатистика\n/stats — текущая неделя + месяц\n/stats_prevweek — прошлая неделя\n/stats_prevmonth — прошлый месяц\n/statsMMYY — месяц по коду (например /stats1125)\n\nСервис\n/hard on|off — жёсткий режим\n/start\n/help",
	)
	msg.ReplyMarkup = telegram.MainMenu()
	_, _ = h.api.Send(msg)
}
