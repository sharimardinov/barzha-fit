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
		"Команды\n\nДень\n/today — тренировка + прогресс\n/status — статус на сегодня\n/week — план по дням\n/steps — шаги за сегодня\n/meal — добавить еду\n/meals — еда за сегодня\n/undo — удалить последний прием\n\nПлан\n/plan — вставить план\n/planset — вставить план\n/planshow — показать план\n/planday <1-7> — показать день\n\nПрофиль\n/profile — профиль\n/profileset — заполнить профиль\n/targets — цели\n/targetsrefresh — пересчитать цели\n/targetsset kcal <число> — задать калории вручную\n/targetsset steps <число> — задать шаги вручную\n\nИстория\n/streak — серии\n\nСтатистика\n/stats — текущая неделя + месяц\n/stats_prevweek — прошлая неделя\n/stats_prevmonth — прошлый месяц\n/statsMMYY — месяц по коду (например /stats1125)\n\nСервис\n/hard on|off — жёсткий режим\n/start\n/help",
	))
}
