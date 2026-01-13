package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func MainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📅 План на сегодня"),
			tgbotapi.NewKeyboardButton("🍽 Добавить еду"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("👟 Добавить шаги"),
			tgbotapi.NewKeyboardButton("📊 Статистика недели"),
		),
	)
}

func WorkoutButtons() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Сделал", "w:done"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Пропустил", "w:skip"),
		),
	)
}

func MealButtons() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↩️ Undo", "meal:undo"),
		),
	)
}

func PlanNavButtons(day int) tgbotapi.InlineKeyboardMarkup {
	prev := day - 1
	if prev < 1 {
		prev = 7
	}
	next := day + 1
	if next > 7 {
		next = 1
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️", "plan:day:"+itoa(prev)),
			tgbotapi.NewInlineKeyboardButtonData("День "+itoa(day), "noop"),
			tgbotapi.NewInlineKeyboardButtonData("▶️", "plan:day:"+itoa(next)),
		),
	)
}

func StatsButtons() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Неделя", "stats:week"),
			tgbotapi.NewInlineKeyboardButtonData("Месяц", "stats:month"),
		),
	)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
