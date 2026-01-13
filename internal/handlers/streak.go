package handlers

import (
	"barzhafit/internal/service"
	"barzhafit/internal/util"
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Streak struct {
	api     *tgbotapi.BotAPI
	workout *service.WorkoutService
	nut     *service.NutritionService
	tz      string
}

func NewStreak(api *tgbotapi.BotAPI, workout *service.WorkoutService, nut *service.NutritionService, tz string) *Streak {
	return &Streak{api: api, workout: workout, nut: nut, tz: tz}
}

func (h *Streak) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID
	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)

	from := util.DayStart(now.AddDate(0, 0, -60), loc)
	to := util.DayStart(now, loc)

	workoutMap, _ := h.workout.ListByRange(ctx, chatID, util.LocalDateStr(from, loc), util.LocalDateStr(to, loc))
	mealMap, _ := h.nut.SumByRangeDaily(ctx, chatID, from, to.Add(24*time.Hour), h.tz)

	workoutStreak := 0
	for i := 0; i <= 60; i++ {
		d := util.DayStart(now.AddDate(0, 0, -i), loc)
		key := util.LocalDateStr(d, loc)
		status, ok := workoutMap[key]
		if !ok || status != "done" {
			break
		}
		workoutStreak++
	}

	noMealStreak := 0
	for i := 0; i <= 60; i++ {
		d := util.DayStart(now.AddDate(0, 0, -i), loc)
		key := util.LocalDateStr(d, loc)
		kcal := 0
		if dn, ok := mealMap[key]; ok {
			kcal = dn.Kcal
		}
		if kcal > 0 {
			break
		}
		noMealStreak++
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Тренировки подряд: %d\n", workoutStreak))
	b.WriteString(fmt.Sprintf("Дней без еды-логов: %d", noMealStreak))

	_, _ = h.api.Send(tgbotapi.NewMessage(chatID, b.String()))
}
