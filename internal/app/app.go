package app

import (
	"context"
	"fmt"

	"barzhafit/internal/bot"
	"barzhafit/internal/config"
	"barzhafit/internal/service"
	"barzhafit/internal/storage/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type App struct {
	cfg *config.Config
	bot *bot.Bot
	db  closer
}

type closer interface {
	Close()
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("telegram init: %w", err)
	}
	api.Debug = cfg.Debug

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	planRepo := db.NewPlanRepo(pool)
	planSvc := service.NewPlanService(planRepo)

	workoutRepo := db.NewWorkoutRepo(pool)
	workoutSvc := service.NewWorkoutService(workoutRepo)

	b := bot.New(api, planSvc, workoutSvc, cfg.TZ)

	return &App{cfg: cfg, bot: b, db: pool}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.db.Close()
	return a.bot.Run(ctx)
}
