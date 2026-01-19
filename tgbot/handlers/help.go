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
	b.WriteString("Справка\n\n")
	b.WriteString("Команды\n")
	b.WriteString("День: /today, /setmeal, /meals, /undo, /setstep\n")
	b.WriteString("План: /plan, /setplan\n")
	b.WriteString("Профиль и цели: /profile, /profileset, /weight, /targets, /targetsrefresh, /targetsset kcal <число>, /targetsset steps <число>\n")
	b.WriteString("История: /streak\n")
	b.WriteString("Статистика: /stats, /stats_prevweek, /stats_prevmonth, /statsMMYY (например /stats1125)\n")
	b.WriteString("Сервис: /morning on|off, /hard on|off, /start\n\n")

	b.WriteString("Как это работает\n")
	b.WriteString("1) Профиль: /profileset ведет по шагам. Можно обновить вес командой /weight.\n")
	b.WriteString("2) Цели: /targetsrefresh пересчитывает цели из профиля, /targetsset меняет калории или шаги вручную.\n")
	b.WriteString("3) Еда: /setmeal — запись еды одним сообщением; /meals — итог дня; /undo — удалить прием.\n")
	b.WriteString("4) Шаги: /setstep записывает шаги за сегодня.\n")
	b.WriteString("5) Тренировки: /today показывает план на день и позволяет отметить done/skip.\n")
	b.WriteString("6) Статистика: /stats показывает текущую неделю и месяц, есть команды для прошлой недели/месяца.\n")
	b.WriteString("7) Серии: /streak показывает тренировки подряд и серию по еде.\n")
	b.WriteString("8) Напоминания: /morning включает утренние сообщения. Вечером бот может напомнить про белок и шаги.\n")
	b.WriteString("9) Жесткий режим: /hard on — вечером строгие напоминания без мотивации.\n\n")

	b.WriteString("Как считаются цели\n")
	b.WriteString("1) Калории: по профилю и активности (TDEE), затем цель влияет на калории.\n")
	b.WriteString("2) Белок: 2.0 г на кг сухой массы.\n")
	b.WriteString("3) Жир: 0.9 г на кг общей массы.\n")
	b.WriteString("4) Углеводы: остаток от калорий после белка и жира.\n")
	b.WriteString("5) Цель: похудение −20%, баланс 0%, набор +10–15%.\n")
	b.WriteString("6) Шаги: по умолчанию 10 000, можно изменить /targetsset steps <число>.\n\n")

	b.WriteString("Иконки прогресса в /today\n")
	b.WriteString("- 🟢 90–110% от цели, иначе 🔴.\n\n")
	b.WriteString("Команды: см. /start")

	_, _ = h.api.Send(tgbotapi.NewMessage(m.Chat.ID, b.String()))
}
