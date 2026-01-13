package service

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"barzhafit/internal/util"
)

type StatsViewService struct {
	nut     *NutritionService
	workout *WorkoutService
	steps   *StepsService
	targets *TargetsService
	tz      string
}

func NewStatsViewService(nut *NutritionService, workout *WorkoutService, steps *StepsService, targets *TargetsService, tz string) *StatsViewService {
	return &StatsViewService{nut: nut, workout: workout, steps: steps, targets: targets, tz: tz}
}

func (s *StatsViewService) WeekText(ctx context.Context, chatID int64, now time.Time, prev bool) (string, error) {
	loc := util.MustLocation(s.tz)
	weekday := util.Weekday1to7(now)
	weekStart := util.DayStart(now.AddDate(0, 0, -(weekday-1)), loc)
	if prev {
		weekStart = weekStart.AddDate(0, 0, -7)
	}
	weekDays := 7
	if !prev {
		weekDays = weekday
	}
	weekEnd := weekStart.AddDate(0, 0, weekDays-1)

	weekFoodMap, _ := s.nut.SumByRangeDaily(ctx, chatID, weekStart, weekEnd.Add(24*time.Hour), s.tz)
	weekStepsMap, _ := s.steps.ListByRange(ctx, chatID, util.LocalDateStr(weekStart, loc), util.LocalDateStr(weekEnd, loc))

	kcalTarget := 2400
	stepsTarget := 10000
	if tg, ok, _ := s.targets.Get(ctx, chatID); ok {
		kcalTarget = tg.Kcal
		if tg.Steps > 0 {
			stepsTarget = tg.Steps
		}
	}

	foodWeek := make([]bool, 0, weekDays)
	stepsWeek := make([]bool, 0, weekDays)
	lines := make([]string, 0, weekDays+2)
	if prev {
		lines = append(lines, "Прошлая неделя")
	} else {
		lines = append(lines, "Текущая неделя")
	}
	lines = append(lines, "Дата        Ккал  Б   Ж   У   Шаги  Трен")

	for i := 0; i < weekDays; i++ {
		dayStart := util.DayStart(weekStart.AddDate(0, 0, i), loc)
		dayDate := util.LocalDateStr(dayStart, loc)

		kcal, p, f, c := 0, 0, 0, 0
		if dn, ok := weekFoodMap[dayDate]; ok {
			kcal, p, f, c = dn.Kcal, dn.P, dn.F, dn.C
		}

		status, hasWorkout, _ := s.workout.GetStatusByDate(ctx, chatID, dayDate)
		steps, hasSteps := 0, false
		if v, ok := weekStepsMap[dayDate]; ok {
			steps, hasSteps = v, true
		}

		dayName := dayStart.Format("02.01 Mon")
		if kcal == 0 && !hasWorkout && !hasSteps {
			lines = append(lines, fmt.Sprintf("%-11s  —", dayName))
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

		kcalStr := fmtIntOrDash(kcal)
		pStr := fmtIntOrDash(p)
		fStr := fmtIntOrDash(f)
		cStr := fmtIntOrDash(c)
		if kcal == 0 {
			pStr, fStr, cStr = "--", "--", "--"
		}
		stepsStr := fmtIntOrDash(steps)
		if !hasSteps {
			stepsStr = "--"
		}
		lines = append(lines, fmt.Sprintf("%-11s %4s  %2s  %2s  %2s  %5s  %s",
			dayName, kcalStr, pStr, fStr, cStr, stepsStr, workoutIcon))

		foodWeek = append(foodWeek, foodInRange(kcal, kcalTarget))
		stepsWeek = append(stepsWeek, steps >= stepsTarget)
	}

	var b strings.Builder
	b.WriteString("<pre>")
	b.WriteString(html.EscapeString(strings.Join(lines, "\n")))
	b.WriteString("</pre>\n")
	b.WriteString("\nНеделя:\n")
	b.WriteString(fmt.Sprintf("Еда: %s\n", weekEmoji(foodWeek)))
	b.WriteString(fmt.Sprintf("Шаги: %s\n", weekEmoji(stepsWeek)))
	return b.String(), nil
}

func (s *StatsViewService) MonthText(ctx context.Context, chatID int64, now time.Time, monthStart time.Time, full bool) (string, error) {
	loc := util.MustLocation(s.tz)
	monthDays := daysInMonth(monthStart)
	if !full {
		monthDays = int(util.DayStart(now.In(loc), loc).Sub(monthStart).Hours()/24) + 1
	}

	monthEnd := monthStart.AddDate(0, 0, monthDays-1)
	foodMap, _ := s.nut.SumByRangeDaily(ctx, chatID, monthStart, monthEnd.Add(24*time.Hour), s.tz)
	stepsMap, _ := s.steps.ListByRange(ctx, chatID, util.LocalDateStr(monthStart, loc), util.LocalDateStr(monthEnd, loc))

	kcalTarget := 2400
	stepsTarget := 10000
	if tg, ok, _ := s.targets.Get(ctx, chatID); ok {
		kcalTarget = tg.Kcal
		if tg.Steps > 0 {
			stepsTarget = tg.Steps
		}
	}

	monthFoodEmojis := make([]string, 0, monthDays)
	monthStepsEmojis := make([]string, 0, monthDays)
	for i := 0; i < monthDays; i++ {
		dayStart := monthStart.AddDate(0, 0, i)
		dayDate := util.LocalDateStr(dayStart, loc)
		kcal := 0
		if dn, ok := foodMap[dayDate]; ok {
			kcal = dn.Kcal
		}
		steps := 0
		if v, ok := stepsMap[dayDate]; ok {
			steps = v
		}
		monthFoodEmojis = append(monthFoodEmojis, dayEmoji(foodInRange(kcal, kcalTarget)))
		monthStepsEmojis = append(monthStepsEmojis, dayEmoji(steps >= stepsTarget))
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Месяц: %s %d\n", monthName(monthStart.Month()), monthStart.Year()))
	b.WriteString("Еда:\n")
	b.WriteString(monthCalendar(monthStart, monthFoodEmojis))
	b.WriteString("Шаги:\n")
	b.WriteString(monthCalendar(monthStart, monthStepsEmojis))
	return b.String(), nil
}

func fmtIntOrDash(v int) string {
	if v == 0 {
		return "--"
	}
	return fmt.Sprintf("%d", v)
}

func foodInRange(kcal int, target int) bool {
	if kcal == 0 {
		return false
	}
	min := int(float64(target) * 0.9)
	max := int(float64(target) * 1.1)
	return kcal >= min && kcal <= max
}

func dayEmoji(ok bool) string {
	if ok {
		return "🟢"
	}
	return "🔴"
}

func weekEmoji(days []bool) string {
	var b strings.Builder
	for _, ok := range days {
		b.WriteString(dayEmoji(ok))
	}
	return b.String()
}

func monthCalendar(monthStart time.Time, emojis []string) string {
	var b strings.Builder
	offset := util.Weekday1to7(monthStart) - 1
	for i := 0; i < offset; i++ {
		b.WriteString("⚪")
	}
	for i, e := range emojis {
		b.WriteString(e)
		if (i+offset+1)%7 == 0 {
			b.WriteString("\n")
		}
	}
	if (len(emojis)+offset)%7 != 0 {
		b.WriteString("\n")
	}
	return b.String()
}

func monthName(m time.Month) string {
	switch m {
	case time.January:
		return "Январь"
	case time.February:
		return "Февраль"
	case time.March:
		return "Март"
	case time.April:
		return "Апрель"
	case time.May:
		return "Май"
	case time.June:
		return "Июнь"
	case time.July:
		return "Июль"
	case time.August:
		return "Август"
	case time.September:
		return "Сентябрь"
	case time.October:
		return "Октябрь"
	case time.November:
		return "Ноябрь"
	case time.December:
		return "Декабрь"
	default:
		return "Месяц"
	}
}

func daysInMonth(monthStart time.Time) int {
	next := monthStart.AddDate(0, 1, 0)
	return int(next.Sub(monthStart).Hours() / 24)
}
