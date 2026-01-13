package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

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

	// plan
	planRepo := db.NewPlanRepo(pool)
	planSvc := service.NewPlanService(planRepo)

	// workout
	workoutRepo := db.NewWorkoutRepo(pool)
	workoutSvc := service.NewWorkoutService(workoutRepo)

	// users for morning push
	botUsersRepo := db.NewBotUsersRepo(pool)
	botUsersSvc := service.NewBotUsersService(botUsersRepo)

	// profile + targets
	profileRepo := db.NewProfileRepo(pool)
	profileSvc := service.NewProfileService(profileRepo)

	targetsRepo := db.NewTargetsRepo(pool)
	targetsSvc := service.NewTargetsService(targetsRepo, profileRepo)

	aiSvc, err := service.NewAIService()
	if err != nil {
		log.Fatal(err)
	}

	mealRepo := db.NewMealRepo(pool)
	nutSvc := service.NewNutritionService(mealRepo, aiSvc)

	if *morning {
		if err := runMorning(ctx, api, planSvc, workoutSvc, botUsersSvc, cfg.TZ); err != nil {
			log.Fatalf("morning failed: %v", err)
		}
		return
	}

	b := bot.New(api, planSvc, workoutSvc, botUsersSvc, cfg.TZ, profileSvc, targetsSvc, nutSvc, aiSvc, pool)
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
	dayDate := util.LocalDateStr(now, loc)

	chatIDs, err := users.ListEnabled(ctx)
	if err != nil {
		return err
	}

	sent := 0
	for _, chatID := range chatIDs {
		planText, err := plan.Get(ctx, chatID)
		if err != nil {
			continue
		}

		cycleDay := util.Weekday1to7(now)

		days := service.SplitPlanByDays(planText)
		block := strings.TrimSpace(days[cycleDay])
		if block == "" {
			block = "План: день не найден"
		}

		status, has, _ := workout.GetStatusByDate(ctx, chatID, dayDate)
		st := "—"
		if has {
			if status == "done" {
				st = "✅"
			} else if status == "skip" {
				st = "❌"
			}
		}

		text := fmt.Sprintf("Тренировка дня\nДень цикла %d\n\n%s\n\nОтметка: %s", cycleDay, block, st)
		msg := tgbotapi.NewMessage(chatID, text)

		if _, err := api.Send(msg); err != nil {
			continue
		}
		time.Sleep(60 * time.Millisecond)
		sent++
	}

	log.Printf("morning: sent=%d", sent)
	return nil
}
