package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	profileRepo := db.NewProfileRepo(pool)
	profileSvc := service.NewProfileService(profileRepo)

	targetsRepo := db.NewTargetsRepo(pool)
	targetsSvc := service.NewTargetsService(targetsRepo, profileRepo)

	trainingProfileRepo := db.NewTrainingProfileRepo(pool)
	trainingProfileSvc := service.NewTrainingProfileService(trainingProfileRepo)

	mealRepo := db.NewMealRepo(pool)
	nutAI := service.NewNutritionAI(aiClient)
	nutSvc := service.NewNutritionService(mealRepo, nutAI)

	stepsRepo := db.NewStepsRepo(pool)
	stepsSvc := service.NewStepsService(stepsRepo)

	workoutPlanRepo := db.NewWorkoutPlanRepo(pool)
	workoutSessionRepo := db.NewWorkoutSessionRepo(pool)
	workoutSetRepo := db.NewWorkoutSetRepo(pool)
	workoutTimerSvc := service.NewWorkoutTimerService(workoutPlanRepo, workoutSessionRepo, workoutSetRepo)
	workoutStatsSvc := service.NewWorkoutInsightsService(workoutSetRepo, aiClient)

	googleAuthRepo := db.NewGoogleAuthRepo(pool)
	googleAuthSvc := service.NewGoogleAuthService(googleAuthRepo)

	webServer := tgapp.NewServer(tgapp.Deps{
		Addr:               cfg.WebAddr,
		BotToken:           cfg.BotToken,
		AuthBotToken:       cfg.AuthBotToken,
		GoogleClientID:     cfg.GoogleClientID,
		GoogleClientSecret: cfg.GoogleClientSecret,
		TZ:                 cfg.TZ,
		Plan:               planSvc,
		Targets:            targetsSvc,
		Nutrition:          nutSvc,
		Steps:              stepsSvc,
		Profile:            profileSvc,
		Training:           trainingProfileSvc,
		WorkoutTimer:       workoutTimerSvc,
		WorkoutStats:       workoutStatsSvc,
		GoogleAuth:         googleAuthSvc,
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
		case upd, ok := <-updates:
			if !ok {
				return errors.New("telegram updates channel closed")
			}
			if upd.PreCheckoutQuery != nil {
				cfg := tgbotapi.PreCheckoutConfig{
					PreCheckoutQueryID: upd.PreCheckoutQuery.ID,
					OK:                 true,
				}
				if _, err := api.Request(cfg); err != nil {
					log.Printf("pre_checkout failed: %v", err)
				}
				continue
			}
			if upd.Message != nil {
				if upd.Message.SuccessfulPayment != nil {
					payment := upd.Message.SuccessfulPayment
					log.Printf("stars payment: user=%d amount=%d currency=%s payload=%s charge_id=%s", upd.Message.Chat.ID, payment.TotalAmount, payment.Currency, payment.InvoicePayload, payment.TelegramPaymentChargeID)
					msg := tgbotapi.NewMessage(upd.Message.Chat.ID, "Спасибо за поддержку! ⭐")
					if _, err := api.Send(msg); err != nil {
						log.Printf("telegram send failed: chat_id=%d err=%v", upd.Message.Chat.ID, err)
					}
					continue
				}
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
