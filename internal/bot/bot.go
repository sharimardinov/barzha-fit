package bot

import (
	"context"
	"fmt"
	"log"

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
		db:        db,
	}
	b.registerRoutes()
	return b
}

func (b *Bot) registerRoutes() {
	start := handlers.NewStart(b.api, b.users)
	meal := handlers.NewMeal(b.api, b.state, b.nutrition, b.tz)
	plan := handlers.NewPlan(b.api, b.state, b.plan)
	morning := handlers.NewMorning(b.api, b.users)
	today := handlers.NewToday(b.api, b.plan, b.workout, b.targets, b.nutrition, b.tz)
	week := handlers.NewWeek(b.api, b.plan, b.tz)
	profile := handlers.NewProfile(b.api, b.profile, b.targets)
	targets := handlers.NewTargets(b.api, b.targets)
	meals := handlers.NewMeals(b.api, b.nutrition, b.tz)
	undo := handlers.NewUndo(b.api, b.nutrition)
	stats := handlers.NewStats(b.api, b.nutrition, b.workout, b.tz)
	debug := handlers.NewDebugMeals(b.api, b.db)

	b.router.Handle("/start", start.Handle)
	b.router.Handle("/today", today.Handle)
	b.router.Handle("/meal", meal.Handle)
	b.router.Handle("/plan", plan.Handle)
	b.router.Handle("/week", week.Handle)
	b.router.Handle("/morning", morning.Handle)
	b.router.Handle("/profile", profile.Handle)
	b.router.Handle("/targets", targets.Handle)
	b.router.Handle("/meals", meals.Handle)
	b.router.Handle("/undo", undo.Handle)
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
				b.reply(upd.Message.Chat.ID, "Не понял команду. /today")
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
			b.reply(chatID, "Записал, но AI упал (сохранил как 0).")
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

	default:
		b.state.Clear(chatID)
		return false
	}
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
