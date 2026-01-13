package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"barzhafit/internal/service"
	"barzhafit/internal/util"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Week struct {
	api   *tgbotapi.BotAPI
	plan  *service.PlanService
	nut   *service.NutritionService
	steps *service.StepsService
	tz    string
}

func NewWeek(api *tgbotapi.BotAPI, plan *service.PlanService, nut *service.NutritionService, steps *service.StepsService, tz string) *Week {
	return &Week{api: api, plan: plan, nut: nut, steps: steps, tz: tz}
}

func (h *Week) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID

	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)
	today := util.Weekday1to7(now) // Пн=1..Вс=7
	weekStart := util.DayStart(now.AddDate(0, 0, -(today-1)), loc)

	planText, err := h.plan.Get(ctx, chatID)
	if err != nil {
		h.api.Send(tgbotapi.NewMessage(chatID, "Нет плана. Сначала /plan"))
		return
	}

	days := service.SplitPlanByDays(planText)

	var b strings.Builder
	b.WriteString("Неделя:\n\n")

	for d := 1; d <= 7; d++ {
		prefix := "  "
		if d == today {
			prefix = "👉"
		}
		block := strings.TrimSpace(days[d])
		if block == "" {
			block = "(пусто)"
		}
		dayStart := weekStart.AddDate(0, 0, d-1)
		dayEnd := dayStart.Add(24 * time.Hour)
		kcal, p, f, c, err := h.nut.SumByDay(ctx, chatID, dayStart, dayEnd)

		foodLine := "Еда: —"
		if err == nil && kcal > 0 {
			foodLine = fmt.Sprintf("Еда: %d ккал (Б%d Ж%d У%d)", kcal, p, f, c)
		}

		dayDate := util.LocalDateStr(dayStart, loc)
		steps, hasSteps, _ := h.steps.GetByDate(ctx, chatID, dayDate)
		stepsLine := "Шаги: —"
		if hasSteps {
			stepsLine = fmt.Sprintf("Шаги: %d / 10000", steps)
		}
		// чуть компактнее: отделяем дни линией
		b.WriteString(fmt.Sprintf("%s День %d\n%s\n%s\n%s\n\n", prefix, d, block, foodLine, stepsLine))
	}

	msg := tgbotapi.NewMessage(chatID, b.String())
	h.api.Send(msg)
}
