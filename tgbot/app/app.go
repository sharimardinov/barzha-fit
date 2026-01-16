package app

import (
	"context"
	"fmt"

	"barzhafit/backend/config"
	"barzhafit/backend/service"
	"barzhafit/backend/storage/db"
	"barzhafit/tgbot"

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

	aiClient, err := service.NewAIClient()
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
	nutAI := service.NewNutritionAI(aiClient)
	nutSvc := service.NewNutritionService(mealRepo, nutAI)

	// steps
	stepsRepo := db.NewStepsRepo(pool)
	stepsSvc := service.NewStepsService(stepsRepo)

	// ИСПРАВЛЕНО: добавлены profileSvc, targetsSvc, pool
	activityAI := service.NewActivityAI(aiClient)
	b := bot.New(api, planSvc, workoutSvc, botUsersSvc, cfg.TZ, profileSvc, targetsSvc, nutSvc, stepsSvc, activityAI)

	return &App{cfg: cfg, bot: b, db: pool}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.db.Close()
	return a.bot.Run(ctx)
}
