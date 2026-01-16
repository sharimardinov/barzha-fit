package handlers

import (
	"barzhafit/backend/domain"
	"barzhafit/backend/service"
	"barzhafit/backend/util"
	"barzhafit/bot/telegram"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Meal struct {
	api   *tgbotapi.BotAPI
	state domain.StateSetter
	nut   *service.NutritionService
	tz    string
}

func NewMeal(api *tgbotapi.BotAPI, state domain.StateSetter, nut *service.NutritionService, tz string) *Meal {
	return &Meal{api: api, state: state, nut: nut, tz: tz}
}

func (h *Meal) Handle(m *tgbotapi.Message) {
	chatID := m.Chat.ID
	ctx := context.Background()

	// поддерживаем /setmeal <текст>
	args := strings.TrimSpace(m.CommandArguments())
	if args != "" {
		loc := util.MustLocation(h.tz)
		now := util.NowIn(loc)

		meal, err := h.nut.AddMealFromText(ctx, chatID, now, args)
		if err != nil {
			log.Printf("ERROR /setmeal: chatID=%d err=%v", chatID, err)
			if errors.Is(err, service.ErrNutritionAI) {
				_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Записал, но AI упал (сохранил как 0)."))
				return
			}
			_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Не в силах сохранить..."))
			return
		}

		if meal.Kcal == 0 && meal.ProteinG == 0 && meal.FatG == 0 && meal.CarbsG == 0 {
			_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Не в силах распознать КБЖУ, давай ка ещё разок (если нужно — /undo)"))
			return
		}
		msg := tgbotapi.NewMessage(chatID,
			fmt.Sprintf("Ок, записал:\n%dkcal (Б%d Ж%d У%d)\n%s",
				meal.Kcal, meal.ProteinG, meal.FatG, meal.CarbsG, meal.Text),
		)
		msg.ReplyMarkup = telegram.MealButtons()
		_, _ = h.api.Send(msg)
		return
	}

	// обычный режим: /setmeal -> ждём следующий текст
	h.state.Set(chatID, domain.StateWaitMealText)
	_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Напиши одним сообщением что ел. Лучше по одному приёму для точности. Пример: \"2 яйца, рис в сухом виде 100г, одна грудка курицы\""))
}
