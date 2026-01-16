package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"barzhafit/backend/util"
)

type PlanViewService struct {
	plan  *PlanService
	nut   *NutritionService
	steps *StepsService
	tz    string
}

func NewPlanViewService(plan *PlanService, nut *NutritionService, steps *StepsService, tz string) *PlanViewService {
	return &PlanViewService{plan: plan, nut: nut, steps: steps, tz: tz}
}

func (s *PlanViewService) DayText(ctx context.Context, chatID int64, day int, now time.Time) (string, error) {
	planText, err := s.plan.Get(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("no plan")
	}

	days := SplitPlanByDays(planText)
	block := strings.TrimSpace(days[day])
	if block == "" {
		block = "(пусто)"
	}

	loc := util.MustLocation(s.tz)
	weekday := util.Weekday1to7(now)
	weekStart := util.DayStart(now.AddDate(0, 0, -(weekday-1)), loc)
	dayStart := weekStart.AddDate(0, 0, day-1)
	dayEnd := dayStart.Add(24 * time.Hour)

	kcal, p, f, c, err := s.nut.SumByDay(ctx, chatID, dayStart, dayEnd)
	foodLine := "Еда: —"
	if err == nil && (kcal > 0 || p > 0 || f > 0 || c > 0) {
		foodLine = fmt.Sprintf("Еда: %d ккал (Б%d Ж%d У%d)", kcal, p, f, c)
	}

	dayDate := util.LocalDateStr(dayStart, loc)
	steps, hasSteps, _ := s.steps.GetByDate(ctx, chatID, dayDate)
	stepsLine := "Шаги: —"
	if hasSteps {
		stepsLine = fmt.Sprintf("Шаги: %d", steps)
	}

	return fmt.Sprintf("День %d\n%s\n\n%s\n%s", day, block, foodLine, stepsLine), nil
}
