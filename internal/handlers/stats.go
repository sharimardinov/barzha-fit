package handlers

import (
	"barzhafit/internal/service"
	"barzhafit/internal/util"
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Stats struct {
	api     *tgbotapi.BotAPI
	nut     *service.NutritionService
	workout *service.WorkoutService
	steps   *service.StepsService
	targets *service.TargetsService
	tz      string
}

func NewStats(api *tgbotapi.BotAPI, nut *service.NutritionService, workout *service.WorkoutService, steps *service.StepsService, targets *service.TargetsService, tz string) *Stats {
	return &Stats{api: api, nut: nut, workout: workout, steps: steps, targets: targets, tz: tz}
}

func (h *Stats) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID
	loc := util.MustLocation(h.tz)
	now := util.NowIn(loc)
	cmd := strings.ToLower(strings.TrimSpace(m.Command()))
	args := strings.ToLower(strings.TrimSpace(m.CommandArguments()))
	cmdStats := strings.TrimPrefix(cmd, "stats")
	if strings.HasPrefix(cmdStats, "_") {
		cmdStats = strings.TrimPrefix(cmdStats, "_")
	}

	var b strings.Builder
	kcalTarget := 2400
	stepsTarget := 10000
	if tg, ok, _ := h.targets.Get(ctx, chatID); ok {
		kcalTarget = tg.Kcal
		if tg.Steps > 0 {
			stepsTarget = tg.Steps
		}
	}

	b.WriteString("📊 Статистика\n\n")

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	if args == "prevmonth" || cmdStats == "prevmonth" {
		monthStart = monthStart.AddDate(0, -1, 0)
	}
	monthOverride := false
	if args == "prevmonth" {
		monthOverride = true
	}
	if monthArg, ok := parseMonthArg(args); ok {
		monthStart = time.Date(monthArg.Year(), time.Month(monthArg.Month()), 1, 0, 0, 0, 0, loc)
		monthOverride = true
	}
	if monthArg, ok := parseMonthArg(strings.TrimPrefix(cmd, "stats")); ok {
		monthStart = time.Date(monthArg.Year(), time.Month(monthArg.Month()), 1, 0, 0, 0, 0, loc)
		monthOverride = true
	}
	if cmdStats == "prevmonth" {
		monthOverride = true
	}

	if !monthOverride {
		// Текущая неделя (Пн..Сегодня) или прошлая (Пн..Вс)
		weekday := util.Weekday1to7(now)
		weekStart := util.DayStart(now.AddDate(0, 0, -(weekday-1)), loc)
		prevWeek := args == "prevweek" || args == "weekprev" || cmdStats == "prevweek" || strings.HasPrefix(cmd, "statsprevweek")
		if prevWeek {
			weekStart = weekStart.AddDate(0, 0, -7)
		}
		weekDays := 7
		if !prevWeek {
			weekDays = weekday
		}
		weekEnd := weekStart.AddDate(0, 0, weekDays-1)
		weekFoodMap, _ := h.nut.SumByRangeDaily(ctx, chatID, weekStart, weekEnd.Add(24*time.Hour), h.tz)
		weekStepsMap, _ := h.steps.ListByRange(ctx, chatID, util.LocalDateStr(weekStart, loc), util.LocalDateStr(weekEnd, loc))
		foodWeek := make([]bool, 0, weekDays)
		stepsWeek := make([]bool, 0, weekDays)
		lines := make([]string, 0, weekDays+2)
		if prevWeek {
			lines = append(lines, "Прошлая неделя")
		} else {
			lines = append(lines, "Текущая неделя")
		}
		lines = append(lines, "Дата        Ккал  Б   Ж   У   Шаги  Трен")
		for i := 0; i < weekDays; i++ {
			dayStart := util.DayStart(weekStart.AddDate(0, 0, i), loc)
			dayDate := util.LocalDateStr(dayStart, loc)

			// Еда за день
			kcal, p, f, c := 0, 0, 0, 0
			if dn, ok := weekFoodMap[dayDate]; ok {
				kcal, p, f, c = dn.Kcal, dn.P, dn.F, dn.C
			}

			// Тренировка за день
			status, hasWorkout, _ := h.workout.GetStatusByDate(ctx, chatID, dayDate)

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

			foodOK := foodInRange(kcal, kcalTarget)
			stepsOK := steps >= stepsTarget
			foodWeek = append(foodWeek, foodOK)
			stepsWeek = append(stepsWeek, stepsOK)
		}

		if len(lines) > 0 {
			b.WriteString("<pre>")
			b.WriteString(html.EscapeString(strings.Join(lines, "\n")))
			b.WriteString("</pre>\n")
		}

		b.WriteString("\nНеделя:\n")
		b.WriteString(fmt.Sprintf("Еда: %s\n", weekEmoji(foodWeek)))
		b.WriteString(fmt.Sprintf("Шаги: %s\n", weekEmoji(stepsWeek)))
	}
	monthDays := daysInMonth(monthStart)
	if !monthOverride && args != "prevmonth" && cmdStats != "prevmonth" {
		monthDays = int(util.DayStart(now, loc).Sub(monthStart).Hours()/24) + 1
	}
	monthEnd := monthStart.AddDate(0, 0, monthDays-1)
	monthFoodMap, _ := h.nut.SumByRangeDaily(ctx, chatID, monthStart, monthEnd.Add(24*time.Hour), h.tz)
	monthStepsMap, _ := h.steps.ListByRange(ctx, chatID, util.LocalDateStr(monthStart, loc), util.LocalDateStr(monthEnd, loc))
	monthFoodEmojis := make([]string, 0, monthDays)
	monthStepsEmojis := make([]string, 0, monthDays)
	for i := 0; i < monthDays; i++ {
		dayStart := monthStart.AddDate(0, 0, i)
		dayDate := util.LocalDateStr(dayStart, loc)
		kcal := 0
		if dn, ok := monthFoodMap[dayDate]; ok {
			kcal = dn.Kcal
		}
		steps := 0
		if v, ok := monthStepsMap[dayDate]; ok {
			steps = v
		}

		foodOK := foodInRange(kcal, kcalTarget)
		stepsOK := steps >= stepsTarget
		monthFoodEmojis = append(monthFoodEmojis, dayEmoji(foodOK))
		monthStepsEmojis = append(monthStepsEmojis, dayEmoji(stepsOK))
	}

	b.WriteString(fmt.Sprintf("\nМесяц: %s %d\n", monthName(monthStart.Month()), monthStart.Year()))
	b.WriteString("Еда:\n")
	b.WriteString(monthCalendar(monthStart, monthFoodEmojis))
	b.WriteString("Шаги:\n")
	b.WriteString(monthCalendar(monthStart, monthStepsEmojis))

	msg := tgbotapi.NewMessage(chatID, b.String())
	msg.ParseMode = tgbotapi.ModeHTML
	_, _ = h.api.Send(msg)
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

func daysInMonth(monthStart time.Time) int {
	next := monthStart.AddDate(0, 1, 0)
	return int(next.Sub(monthStart).Hours() / 24)
}

type monthArg struct {
	month int
	year  int
}

func (m monthArg) Month() int { return m.month }
func (m monthArg) Year() int  { return m.year }

func parseMonthArg(s string) (monthArg, bool) {
	if len(s) != 4 {
		return monthArg{}, false
	}
	mm, err := strconv.Atoi(s[:2])
	if err != nil || mm < 1 || mm > 12 {
		return monthArg{}, false
	}
	yy, err := strconv.Atoi(s[2:])
	if err != nil {
		return monthArg{}, false
	}
	return monthArg{month: mm, year: 2000 + yy}, true
}

func fmtIntOrDash(v int) string {
	if v == 0 {
		return "--"
	}
	return strconv.Itoa(v)
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
