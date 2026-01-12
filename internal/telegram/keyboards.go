package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func WorkoutButtons() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Сделал", "w:done"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Пропустил", "w:skip"),
		),
	)
}
