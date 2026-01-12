package bot

import (
	"barzhafit/internal/domain"
	"barzhafit/internal/handlers"
	"barzhafit/internal/service"
	"barzhafit/internal/util"
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	router  *Router
	state   *StateStore
	plan    *service.PlanService
	workout *service.WorkoutService
	tz      string
}

func New(api *tgbotapi.BotAPI,
	plan *service.PlanService,
	workout *service.WorkoutService,
	tz string) *Bot {
	b := &Bot{
		api:     api,
		router:  NewRouter(),
		state:   NewStateStore(),
		plan:    plan,
		workout: workout,
		tz:      tz,
	}
	b.registerRoutes()
	return b
}

func (b *Bot) registerRoutes() {
	start := handlers.NewStart(b.api)
	today := handlers.NewToday(b.api, b.plan, b.workout, b.tz)
	meal := handlers.NewMeal(b.api, b.state)
	plan := handlers.NewPlan(b.api, b.state, b.plan)

	b.router.Handle("/start", start.Handle)
	b.router.Handle("/today", today.Handle)
	b.router.Handle("/meal", meal.Handle)
	b.router.Handle("/plan", plan.Handle)
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
		// пока просто подтверждаем, позже тут будет AI + сохранение
		text := m.Text
		b.state.Clear(chatID)
		b.reply(chatID, "Ок, записал приём пищи (пока без подсчёта):\n"+text)
		return true

	case domain.StateWaitPlanText:
		planText := m.Text
		b.state.Clear(chatID)

		if err := b.plan.Save(context.Background(), chatID, planText); err != nil {
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
	day := util.Weekday1to7(now)

	if err := b.workout.MarkToday(context.Background(), chatID, now, day, status); err != nil {
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
