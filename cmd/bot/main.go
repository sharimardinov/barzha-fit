package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	_ "time"

	"barzhafit/internal/bot"
	"barzhafit/internal/config"
	"barzhafit/internal/service"
	"barzhafit/internal/storage/db"
	"barzhafit/internal/util"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	morning := flag.Bool("morning", false, "send morning workout to all users and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("telegram init: %v", err)
	}
	api.Debug = cfg.Debug

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// services
	planRepo := db.NewPlanRepo(pool)
	planSvc := service.NewPlanService(planRepo)

	workoutRepo := db.NewWorkoutRepo(pool)
	workoutSvc := service.NewWorkoutService(workoutRepo)

	botUsersRepo := db.NewBotUsersRepo(pool)
	botUsersSvc := service.NewBotUsersService(botUsersRepo)

	if *morning {
		if err := runMorning(ctx, api, planSvc, workoutSvc, botUsersSvc, cfg.TZ); err != nil {
			log.Fatalf("morning failed: %v", err)
		}
		return
	}

	// regular bot
	b := bot.New(api, planSvc, workoutSvc, botUsersSvc, cfg.TZ)
	if err := b.Run(ctx); err != nil {
		log.Fatal(err)
	}
}

func runMorning(
	ctx context.Context,
	api *tgbotapi.BotAPI,
	plan *service.PlanService,
	workout *service.WorkoutService,
	users *service.BotUsersService,
	tz string,
) error {
	loc := util.MustLocation(tz)
	now := util.NowIn(loc)
	day := util.Weekday1to7(now)

	chatIDs, err := users.List(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}

	sent := 0
	for _, chatID := range chatIDs {
		planText, err := plan.Get(ctx, chatID)
		if err != nil {
			// у юзера нет плана — просто пропускаем, без спама
			continue
		}

		block := strings.TrimSpace(service.SplitPlanByDays(planText)[day])
		if block == "" {
			block = "Пусто"
		}

		status, has, err := workout.GetStatusToday(ctx, chatID, now)
		if err != nil {
			log.Printf("morning: status read failed chat_id=%d err=%v", chatID, err)
			continue
		}

		st := "—"
		if has {
			if status == "done" {
				st = "✅"
			} else if status == "skip" {
				st = "❌"
			}
		}

		text := fmt.Sprintf("Тренировка дня\nДень %d\n\n%s\n\nОтметка: %s", day, block, st)

		msg := tgbotapi.NewMessage(chatID, text)
		if _, err := api.Send(msg); err != nil {
			log.Printf("morning: send failed chat_id=%d err=%v", chatID, err)
			continue
		}
		sent++
	}

	log.Printf("morning: done. day=%d users=%d sent=%d", day, len(chatIDs), sent)
	return nil
}
