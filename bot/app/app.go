package app

import (
	"context"
	"fmt"

	"barzhafit/backend/config"
	"barzhafit/backend/service"
	"barzhafit/backend/storage/db"
	"barzhafit/bot"

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

	aiSvc, err := service.NewAIService()
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

	// plan
	planRepo := db.NewPlanRepo(pool)
	planSvc := service.NewPlanService(planRepo)

	// workout (calendar mode)
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

	// nutrition
	mealRepo := db.NewMealRepo(pool)
	nutSvc := service.NewNutritionService(mealRepo, aiSvc)

	// steps
	stepsRepo := db.NewStepsRepo(pool)
	stepsSvc := service.NewStepsService(stepsRepo)

	// ИСПРАВЛЕНО: добавлены profileSvc, targetsSvc, pool
	b := bot.New(api, planSvc, workoutSvc, botUsersSvc, cfg.TZ, profileSvc, targetsSvc, nutSvc, stepsSvc, aiSvc, pool)

	return &App{cfg: cfg, bot: b, db: pool}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.db.Close()
	return a.bot.Run(ctx)
}
