package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"barzhafit/backend/config"
	"barzhafit/backend/service"
	"barzhafit/backend/storage/db"
	"barzhafit/tgapp"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
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

	aiClient, err := service.NewAIClient()
	if err != nil {
		log.Fatal(err)
	}

	planRepo := db.NewPlanRepo(pool)
	planSvc := service.NewPlanService(planRepo)

	workoutRepo := db.NewWorkoutRepo(pool)
	workoutSvc := service.NewWorkoutService(workoutRepo)

	profileRepo := db.NewProfileRepo(pool)
	profileSvc := service.NewProfileService(profileRepo)

	targetsRepo := db.NewTargetsRepo(pool)
	targetsSvc := service.NewTargetsService(targetsRepo, profileRepo)

	trainingProfileRepo := db.NewTrainingProfileRepo(pool)
	trainingProfileSvc := service.NewTrainingProfileService(trainingProfileRepo)

	appUserRepo := db.NewAppUserRepo(pool)
	trainingInputRepo := db.NewTrainingInputRepo(pool)
	trainingInputSvc := service.NewTrainingInputService(trainingInputRepo, appUserRepo)

	templateRepo := db.NewProgramTemplateRepo(pool)
	exerciseRepo := db.NewExerciseRepo(pool)
	userProgramRepo := db.NewUserProgramRepo(pool)
	trainingProgramSvc := service.NewTrainingProgramService(
		appUserRepo,
		trainingInputRepo,
		templateRepo,
		exerciseRepo,
		userProgramRepo,
	)

	injuryTypeRepo := db.NewInjuryTypeRepo(pool)
	injuryTypeSvc := service.NewInjuryTypeService(injuryTypeRepo)

	mealRepo := db.NewMealRepo(pool)
	nutAI := service.NewNutritionAI(aiClient)
	nutSvc := service.NewNutritionService(mealRepo, nutAI)

	stepsRepo := db.NewStepsRepo(pool)
	stepsSvc := service.NewStepsService(stepsRepo)

	activityAI := service.NewActivityAI(aiClient)
	workoutPlanRepo := db.NewWorkoutPlanRepo(pool)
	workoutSessionRepo := db.NewWorkoutSessionRepo(pool)
	workoutSetRepo := db.NewWorkoutSetRepo(pool)
	workoutTimerSvc := service.NewWorkoutTimerService(workoutPlanRepo, workoutSessionRepo, workoutSetRepo)
	workoutStatsSvc := service.NewWorkoutStatsService(workoutSetRepo)

	webServer := tgapp.NewServer(tgapp.Deps{
		Addr:          cfg.WebAddr,
		BotToken:      cfg.BotToken,
		AuthBotToken:  cfg.AuthBotToken,
		TZ:            cfg.TZ,
		Plan:          planSvc,
		Workout:       workoutSvc,
		Targets:       targetsSvc,
		Nutrition:     nutSvc,
		Steps:         stepsSvc,
		Profile:       profileSvc,
		Training:      trainingProfileSvc,
		Inputs:        trainingInputSvc,
		Programs:      trainingProgramSvc,
		Injuries:      injuryTypeSvc,
		Activity:      activityAI,
		WorkoutTimer:  workoutTimerSvc,
		StrengthStats: workoutStatsSvc,
	})
	go func() {
		if err := webServer.ListenAndServe(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("miniapp server error: %v", err)
		}
	}()

	if err := runLinkBot(ctx, api); err != nil {
		log.Fatal(err)
	}
}

func runLinkBot(ctx context.Context, api *tgbotapi.BotAPI) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := api.GetUpdatesChan(u)

	log.Printf("bot started: @%s", api.Self.UserName)

	for {
		select {
		case <-ctx.Done():
			return nil
		case upd := <-updates:
			if upd.Message != nil {
				sendAppLink(api, upd.Message.Chat.ID)
				continue
			}
			if upd.CallbackQuery != nil {
				if upd.CallbackQuery.Message != nil {
					sendAppLink(api, upd.CallbackQuery.Message.Chat.ID)
				}
				_, _ = api.Request(tgbotapi.NewCallback(upd.CallbackQuery.ID, ""))
			}
		}
	}
}

func sendAppLink(api *tgbotapi.BotAPI, chatID int64) {
	link := "https://t.me/" + api.Self.UserName + "?startapp"
	msg := tgbotapi.NewMessage(chatID, "Открыть приложение:\n"+link)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Открыть в полный экран", link),
		),
	)
	if _, err := api.Send(msg); err != nil {
		log.Printf("telegram send failed: chat_id=%d err=%v", chatID, err)
	}
}
