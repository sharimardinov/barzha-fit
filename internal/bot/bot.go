package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"barzhafit/internal/domain"
	"barzhafit/internal/handlers"
	"barzhafit/internal/service"
	"barzhafit/internal/util"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Bot struct {
	api       *tgbotapi.BotAPI
	router    *Router
	state     *StateStore
	plan      *service.PlanService
	workout   *service.WorkoutService
	users     *service.BotUsersService
	tz        string
	profile   *service.ProfileService
	targets   *service.TargetsService
	nutrition *service.NutritionService
	steps     *service.StepsService
	ai        *service.AIService
	drafts    *service.ProfileDraftStore
	db        *pgxpool.Pool
}

func New(
	api *tgbotapi.BotAPI,
	plan *service.PlanService,
	workout *service.WorkoutService,
	users *service.BotUsersService,
	tz string,
	profile *service.ProfileService,
	targets *service.TargetsService,
	nutrition *service.NutritionService,
	steps *service.StepsService,
	ai *service.AIService,
	db *pgxpool.Pool,
) *Bot {
	b := &Bot{
		api:       api,
		router:    NewRouter(),
		state:     NewStateStore(),
		plan:      plan,
		workout:   workout,
		users:     users,
		tz:        tz,
		profile:   profile,
		targets:   targets,
		nutrition: nutrition,
		steps:     steps,
		ai:        ai,
		drafts:    service.NewProfileDraftStore(),
		db:        db,
	}
	b.registerRoutes()
	return b
}

func (b *Bot) registerRoutes() {
	start := handlers.NewStart(b.api, b.users)
	help := handlers.NewHelp(b.api)
	meal := handlers.NewMeal(b.api, b.state, b.nutrition, b.tz)
	plan := handlers.NewPlan(b.api, b.state, b.plan, b.nutrition, b.steps, b.tz)
	morning := handlers.NewMorning(b.api, b.users)
	today := handlers.NewToday(b.api, b.plan, b.workout, b.targets, b.nutrition, b.tz)
	profile := handlers.NewProfile(b.api, b.state, b.drafts, b.profile, b.targets, b.plan, b.ai)
	targets := handlers.NewTargets(b.api, b.targets)
	meals := handlers.NewMeals(b.api, b.nutrition, b.tz)
	undo := handlers.NewUndo(b.api, b.nutrition)
	stats := handlers.NewStats(b.api, b.nutrition, b.workout, b.steps, b.targets, b.tz)
	steps := handlers.NewSteps(b.api, b.state, b.steps, b.tz)
	status := handlers.NewStatus(b.api, b.workout, b.targets, b.nutrition, b.steps, b.tz)
	streak := handlers.NewStreak(b.api, b.workout, b.nutrition, b.tz)
	hard := handlers.NewHard(b.api, b.users)
	debug := handlers.NewDebugMeals(b.api, b.db)

	b.router.Handle("/start", start.Handle)
	b.router.Handle("/help", help.Handle)
	b.router.Handle("/today", today.Handle)
	b.router.Handle("/setmeal", meal.Handle)
	b.router.Handle("/plan", plan.Handle)
	b.router.Handle("/setplan", plan.Handle)
	b.router.Handle("/morning", morning.Handle)
	b.router.Handle("/hard", hard.Handle)
	b.router.Handle("/profile", profile.Handle)
	b.router.Handle("/profileset", profile.Handle)
	b.router.Handle("/targets", targets.Handle)
	b.router.Handle("/targetsrefresh", targets.Handle)
	b.router.Handle("/targetsset", targets.Handle)
	b.router.Handle("/meals", meals.Handle)
	b.router.Handle("/setstep", steps.Handle)
	b.router.Handle("/undo", undo.Handle)
	b.router.Handle("/status", status.Handle)
	b.router.Handle("/streak", streak.Handle)
	b.router.Handle("/stats", stats.Handle)
	b.router.Handle("debugmeals", debug.Handle)
}

func (b *Bot) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.api.GetUpdatesChan(u)

	log.Printf("bot started: @%s", b.api.Self.UserName)

	for {
		select {
		case <-ctx.Done():
			return nil
		case upd := <-updates:

			if upd.CallbackQuery != nil {
				b.handleCallback(upd.CallbackQuery)
				continue
			}

			if upd.Message == nil {
				continue
			}

			if b.handleState(upd.Message) {
				continue
			}
			if ok := b.router.Dispatch(upd.Message); ok {
				continue
			}

			if upd.Message.IsCommand() {
				b.reply(upd.Message.Chat.ID, "Не понял команду. /help")
			}
		}
	}
}

func (b *Bot) reply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	if err != nil {
		fmt.Println("send error:", err)
	}
}

func (b *Bot) handleState(m *tgbotapi.Message) bool {
	chatID := m.Chat.ID
	st := b.state.Get(chatID)

	if st == domain.StateNone {
		return false
	}
	if m.IsCommand() {
		return false
	}

	switch st {
	case domain.StateWaitMealText:
		text := m.Text
		b.state.Clear(chatID)

		loc := util.MustLocation(b.tz)
		now := util.NowIn(loc)

		log.Printf("DEBUG saving meal: chatID=%d now=%s text=%s",
			chatID, now.Format("2006-01-02 15:04:05 MST"), text)

		meal, err := b.nutrition.AddMealFromText(context.Background(), chatID, now, text)
		if err != nil {
			log.Printf("ERROR saving meal: chatID=%d err=%v", chatID, err)
			if errors.Is(err, service.ErrNutritionAI) {
				b.reply(chatID, "Записал, но AI упал (сохранил как 0).")
				return true
			}
			b.reply(chatID, "Не удалось сохранить прием пищи.")
			return true
		}

		log.Printf("DEBUG meal saved: chatID=%d id=%d eatenAt=%s kcal=%d",
			chatID, meal.ID, meal.EatenAt.Format("2006-01-02 15:04:05 MST"), meal.Kcal)

		b.reply(chatID, fmt.Sprintf("Ок, записал:\n%dkcal (Б%d Ж%d У%d)\n%s",
			meal.Kcal, meal.ProteinG, meal.FatG, meal.CarbsG, meal.Text))
		return true

	case domain.StateWaitPlanText:
		planText := m.Text
		b.state.Clear(chatID)

		if err := b.plan.Save(context.Background(), chatID, planText); err != nil {
			log.Printf("plan save failed: chat_id=%d err=%v", chatID, err)
			b.reply(chatID, "Ошибка сохранения плана")
			return true
		}

		b.reply(chatID, "План сохранён")
		return true

	case domain.StateWaitProfileSex:
		sex := strings.ToLower(strings.TrimSpace(m.Text))
		if sex == "м" || sex == "m" || sex == "male" {
			sex = "m"
		} else if sex == "ж" || sex == "f" || sex == "female" {
			sex = "f"
		} else {
			b.reply(chatID, "Введи пол: м или ж.")
			return true
		}
		if !b.drafts.Update(chatID, func(d *service.ProfileDraft) {
			d.Sex = sex
		}) {
			b.state.Clear(chatID)
			b.reply(chatID, "Сначала /profileset")
			return true
		}
		b.state.Set(chatID, domain.StateWaitProfileHeight)
		b.reply(chatID, "Теперь рост в см, например 180.")
		return true

	case domain.StateWaitProfileHeight:
		height, ok := parseIntInRange(strings.TrimSpace(m.Text), 50, 260)
		if !ok {
			b.reply(chatID, "Введи рост в см, например 180.")
			return true
		}
		if !b.drafts.Update(chatID, func(d *service.ProfileDraft) {
			d.HeightCM = height
		}) {
			b.state.Clear(chatID)
			b.reply(chatID, "Сначала /profileset")
			return true
		}
		b.state.Set(chatID, domain.StateWaitProfileWeight)
		b.reply(chatID, "Теперь вес в кг, например 82.5.")
		return true

	case domain.StateWaitProfileWeight:
		weight, ok := parseFloatInRange(strings.TrimSpace(m.Text), 20, 400)
		if !ok {
			b.reply(chatID, "Введи вес в кг, например 82.5.")
			return true
		}
		if !b.drafts.Update(chatID, func(d *service.ProfileDraft) {
			d.WeightKG = weight
		}) {
			b.state.Clear(chatID)
			b.reply(chatID, "Сначала /profileset")
			return true
		}
		b.state.Set(chatID, domain.StateWaitProfileBodyFat)
		b.reply(chatID, "Процент жира, например 15.")
		return true

	case domain.StateWaitProfileBodyFat:
		bf, ok := parseFloatInRange(strings.TrimSpace(m.Text), 3, 80)
		if !ok {
			b.reply(chatID, "Введи процент жира, например 15.")
			return true
		}
		if !b.drafts.Update(chatID, func(d *service.ProfileDraft) {
			d.BodyFatPct = bf
		}) {
			b.state.Clear(chatID)
			b.reply(chatID, "Сначала /profileset")
			return true
		}
		b.state.Set(chatID, domain.StateWaitProfileAge)
		b.reply(chatID, "Возраст, например 30.")
		return true

	case domain.StateWaitStepsCount:
		steps, ok := parseIntInRange(strings.TrimSpace(m.Text), 0, 100000)
		if !ok {
			b.reply(chatID, "Напиши количество шагов числом, например 8500.")
			return true
		}
		loc := util.MustLocation(b.tz)
		dayDate := util.LocalDateStr(util.NowIn(loc), loc)
		if err := b.steps.SetSteps(context.Background(), chatID, dayDate, steps); err != nil {
			log.Printf("steps save failed: chat_id=%d err=%v", chatID, err)
			b.reply(chatID, "Не удалось сохранить шаги.")
			return true
		}
		b.reply(chatID, fmt.Sprintf("Ок, записал: %d шагов", steps))
		return true

	case domain.StateWaitProfileAge:
		age, ok := parseIntInRange(strings.TrimSpace(m.Text), 10, 100)
		if !ok {
			b.reply(chatID, "Введи возраст, например 30.")
			return true
		}

		if !b.drafts.Update(chatID, func(d *service.ProfileDraft) {
			d.Age = age
		}) {
			b.state.Clear(chatID)
			b.reply(chatID, "Сначала /profileset")
			return true
		}

		draft, ch, ok := b.drafts.Snapshot(chatID)
		if !ok {
			b.state.Clear(chatID)
			b.reply(chatID, "Сначала /profileset")
			return true
		}

		if ch != nil && !draft.ActivityReady {
			select {
			case <-ch:
				draft, _, _ = b.drafts.Snapshot(chatID)
			case <-time.After(2 * time.Second):
			}
		}

		p := domain.Profile{
			ChatID:     chatID,
			Sex:        draft.Sex,
			HeightCM:   draft.HeightCM,
			WeightKG:   draft.WeightKG,
			BodyFatPct: draft.BodyFatPct,
			Age:        draft.Age,
			Activity:   draft.Activity,
		}

		if err := b.profile.Save(context.Background(), p); err != nil {
			b.reply(chatID, "Ошибка сохранения профиля.")
			return true
		}

		b.state.Clear(chatID)
		b.drafts.Clear(chatID)

		activityNote := draft.Activity
		if strings.HasPrefix(strings.ToLower(draft.Activity), "ai:") {
			activityNote = strings.TrimPrefix(draft.Activity, "ai:") + " (ai)"
		}

		if draft.ActivityErr != nil {
			activityNote += " (AI не смог оценить)"
		}

		sex := p.Sex
		if sex == "" {
			sex = "—"
		}
		b.reply(chatID, fmt.Sprintf(
			"Профиль сохранён:\nПол: %s\nВозраст: %d\nРост: %d см\nВес: %.1f кг\nЖир: %.1f%%\nАктивность: %s",
			sex, p.Age, p.HeightCM, p.WeightKG, p.BodyFatPct, activityNote,
		))
		return true

	default:
		b.state.Clear(chatID)
		return false
	}
}

func parseIntInRange(s string, min, max int) (int, bool) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	if v < min || v > max {
		return 0, false
	}
	return v, true
}

func parseFloatInRange(s string, min, max float64) (float64, bool) {
	s = strings.ReplaceAll(s, ",", ".")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	if v < min || v > max {
		return 0, false
	}
	return v, true
}

func (b *Bot) handleCallback(q *tgbotapi.CallbackQuery) {
	cfg := tgbotapi.CallbackConfig{CallbackQueryID: q.ID}
	_, _ = b.api.Request(cfg)

	if q.Message == nil || q.Message.Chat == nil {
		return
	}

	chatID := q.Message.Chat.ID
	data := q.Data
	if data != "w:done" && data != "w:skip" {
		return
	}

	status := "skip"
	if data == "w:done" {
		status = "done"
	}

	loc := util.MustLocation(b.tz)
	now := util.NowIn(loc)
	dayDate := util.LocalDateStr(now, loc)

	_, err := b.workout.MarkAndAdvance(context.Background(), chatID, dayDate, status)
	if err != nil {
		log.Printf("workout mark failed: chat_id=%d err=%v", chatID, err)
		b.reply(chatID, "Ошибка сохранения тренировки")
		return
	}

	if status == "done" {
		b.reply(chatID, "Ок, записал: ✅")
	} else {
		b.reply(chatID, "Ок, записал: ❌")
	}
}
