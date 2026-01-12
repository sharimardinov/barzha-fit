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

type Stats struct {
	api     *tgbotapi.BotAPI
	nut     *service.NutritionService
	workout *service.WorkoutService
	tz      string
}

func NewStats(api *tgbotapi.BotAPI, nut *service.NutritionService, workout *service.WorkoutService, tz string) *Stats {
	return &Stats{api: api, nut: nut, workout: workout, tz: tz}
}

func (h *Stats) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID
	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)

	var b strings.Builder
	b.WriteString("📊 Статистика за неделю\n\n")

	// Последние 7 дней
	for i := 6; i >= 0; i-- {
		dayStart := util.DayStart(now.AddDate(0, 0, -i), loc)
		dayEnd := dayStart.Add(24 * time.Hour)

		// Еда за день
		kcal, p, f, c, err := h.nut.SumByDay(ctx, chatID, dayStart, dayEnd)

		// Тренировка за день
		dayDate := util.LocalDateStr(dayStart, util.MustLocation(h.tz))
		status, hasWorkout, _ := h.workout.GetStatusByDate(ctx, chatID, dayDate)

		dayName := dayStart.Format("02.01 Mon")

		if err != nil || (kcal == 0 && !hasWorkout) {
			b.WriteString(fmt.Sprintf("%s: —\n", dayName))
			continue
		}

		workoutIcon := "—"
		if hasWorkout {
			if status == "done" {
				workoutIcon = "✅"
			} else if status == "skip" {
				workoutIcon = "❌"
			}
		}

		if kcal > 0 {
			b.WriteString(fmt.Sprintf("%s: %d ккал (Б%d Ж%d У%d) %s\n",
				dayName, kcal, p, f, c, workoutIcon))
		} else {
			b.WriteString(fmt.Sprintf("%s: тренировка %s\n", dayName, workoutIcon))
		}
	}

	// Итого за неделю
	weekStart := util.DayStart(now.AddDate(0, 0, -6), loc)
	weekEnd := util.DayStart(now, loc).Add(24 * time.Hour)

	totalKcal, totalP, totalF, totalC, err := h.nut.SumByWeek(ctx, chatID, weekStart, weekEnd)
	if err == nil && totalKcal > 0 {
		avgKcal := totalKcal / 7
		b.WriteString(fmt.Sprintf("\n📈 Итого: %d ккал\n", totalKcal))
		b.WriteString(fmt.Sprintf("Среднее: %d ккал/день\n", avgKcal))
		b.WriteString(fmt.Sprintf("Макросы: Б%d Ж%d У%d\n", totalP, totalF, totalC))
	}

	_, _ = h.api.Send(tgbotapi.NewMessage(chatID, b.String()))
}
