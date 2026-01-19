package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"barzhafit/backend/service"
	"barzhafit/backend/util"
	"barzhafit/tgbot/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CallbackService struct {
	workout  *service.WorkoutService
	nut      *service.NutritionService
	planView *service.PlanViewService
	stats    *service.StatsViewService
	tz       string
}

func NewCallbackService(workout *service.WorkoutService, nut *service.NutritionService, planView *service.PlanViewService, stats *service.StatsViewService, tz string) *CallbackService {
	return &CallbackService{workout: workout, nut: nut, planView: planView, stats: stats, tz: tz}
}

type CallbackResult struct {
	Text   string
	Markup *tgbotapi.InlineKeyboardMarkup
	Mode   string
}

func (s *CallbackService) Handle(ctx context.Context, chatID int64, data string) (CallbackResult, bool, error) {
	switch {
	case data == "noop":
		return CallbackResult{}, true, nil
	case data == "w:done" || data == "w:skip":
		status := "skip"
		if data == "w:done" {
			status = "done"
		}

		loc := util.MustLocation(s.tz)
		now := util.NowIn(loc)
		dayDate := util.LocalDateStr(now, loc)

		_, err := s.workout.MarkAndAdvance(ctx, chatID, dayDate, status)
		if err != nil {
			return CallbackResult{}, false, err
		}

		if status == "done" {
			return CallbackResult{Text: "Ок, записал: ✅"}, false, nil
		}
		return CallbackResult{Text: "Ок, записал: ❌"}, false, nil

	case data == "meal:undo":
		ok, err := s.nut.UndoLast(ctx, chatID)
		if err != nil {
			return CallbackResult{}, false, err
		}
		if !ok {
			return CallbackResult{Text: "Нечего удалять."}, false, nil
		}
		return CallbackResult{Text: "Последний прием удален."}, false, nil

	case strings.HasPrefix(data, "meal:del:"):
		idStr := strings.TrimPrefix(data, "meal:del:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return CallbackResult{}, false, nil
		}
		ok, err := s.nut.DeleteByID(ctx, chatID, id)
		if err != nil {
			return CallbackResult{}, false, err
		}
		if !ok {
			return CallbackResult{Text: "Не найдено."}, false, nil
		}
		return CallbackResult{Text: "Приём удалён."}, false, nil

	case strings.HasPrefix(data, "plan:day:"):
		dayStr := strings.TrimPrefix(data, "plan:day:")
		day, err := strconv.Atoi(dayStr)
		if err != nil || day < 1 || day > 7 {
			return CallbackResult{}, false, nil
		}
		loc := util.MustLocation(s.tz)
		now := util.NowIn(loc)
		text, err := s.planView.DayText(ctx, chatID, day, now)
		if err != nil {
			return CallbackResult{Text: "Нет плана. Сначала /setplan"}, false, nil
		}
		markup := telegram.PlanNavButtons(day)
		return CallbackResult{Text: text, Markup: &markup}, false, nil

	case data == "stats:week":
		loc := util.MustLocation(s.tz)
		now := util.NowIn(loc)
		text, err := s.stats.WeekText(ctx, chatID, now, false)
		if err != nil {
			return CallbackResult{}, false, err
		}
		markup := telegram.StatsButtons()
		return CallbackResult{Text: text, Markup: &markup, Mode: tgbotapi.ModeHTML}, false, nil

	case data == "stats:month":
		loc := util.MustLocation(s.tz)
		now := util.NowIn(loc)
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		text, err := s.stats.MonthText(ctx, chatID, now, monthStart, true)
		if err != nil {
			return CallbackResult{}, false, err
		}
		markup := telegram.StatsButtons()
		return CallbackResult{Text: text, Markup: &markup, Mode: tgbotapi.ModeHTML}, false, nil
	}

	return CallbackResult{}, false, nil
}

func (s *CallbackService) ErrorText(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("Ошибка: %v", err)
}
