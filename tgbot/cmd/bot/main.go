package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"barzhafit/backend/config"
	"barzhafit/backend/service"
	"barzhafit/backend/storage/db"
	"barzhafit/backend/util"
	"barzhafit/tgapp"
	bot "barzhafit/tgbot"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	morning := flag.Bool("morning", false, "send morning workout to all users and exit")
	evening := flag.Bool("evening", false, "ask steps from all users and exit")
	day := flag.Bool("day", false, "send day meal reminder to all users and exit")
	weekly := flag.Bool("weekly", false, "send weekly reflection to all users and exit")
	weight := flag.Bool("weight", false, "send monday weight reminder to all users and exit")
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

	trainingProfileRepo := db.NewTrainingProfileRepo(pool)
	trainingProfileSvc := service.NewTrainingProfileService(trainingProfileRepo)
	appUserRepo := db.NewAppUserRepo(pool)
	trainingInputRepo := db.NewTrainingInputRepo(pool)
	trainingInputSvc := service.NewTrainingInputService(trainingInputRepo, appUserRepo)
	templateRepo := db.NewProgramTemplateRepo(pool)
	exerciseRepo := db.NewExerciseRepo(pool)
	userProgramRepo := db.NewUserProgramRepo(pool)
	trainingProgramSvc := service.NewTrainingProgramService(appUserRepo, trainingInputRepo, templateRepo, exerciseRepo, userProgramRepo)
	injuryTypeRepo := db.NewInjuryTypeRepo(pool)
	injuryTypeSvc := service.NewInjuryTypeService(injuryTypeRepo)

	aiClient, err := service.NewAIClient()
	if err != nil {
		log.Fatal(err)
	}

	mealRepo := db.NewMealRepo(pool)
	nutAI := service.NewNutritionAI(aiClient)
	nutSvc := service.NewNutritionService(mealRepo, nutAI)

	stepsRepo := db.NewStepsRepo(pool)
	stepsSvc := service.NewStepsService(stepsRepo)

	planView := service.NewPlanViewService(planSvc, nutSvc, stepsSvc, cfg.TZ)
	statsView := service.NewStatsViewService(nutSvc, workoutSvc, stepsSvc, targetsSvc, cfg.TZ)

	activityAI := service.NewActivityAI(aiClient)
	reflectionAI := service.NewReflectionAI(aiClient)

	if *morning {
		if err := runMorning(ctx, api, planSvc, workoutSvc, botUsersSvc, cfg.TZ); err != nil {
			log.Fatalf("morning failed: %v", err)
		}
		return
	}

	if *day {
		if err := runDay(ctx, api, botUsersSvc, nutSvc, cfg.TZ); err != nil {
			log.Fatalf("day failed: %v", err)
		}
		return
	}

	if *evening {
		if err := runEvening(ctx, api, botUsersSvc, nutSvc, targetsSvc, workoutSvc, stepsSvc, cfg.TZ); err != nil {
			log.Fatalf("evening failed: %v", err)
		}
		return
	}

	if *weekly {
		if err := runWeekly(ctx, api, botUsersSvc, nutSvc, workoutSvc, targetsSvc, reflectionAI, cfg.TZ); err != nil {
			log.Fatalf("weekly failed: %v", err)
		}
		return
	}

	if *weight {
		if err := runWeightReminder(ctx, api, botUsersSvc, profileSvc, cfg.TZ); err != nil {
			log.Fatalf("weight reminder failed: %v", err)
		}
		return
	}

	b := bot.New(api, planSvc, workoutSvc, botUsersSvc, cfg.TZ, profileSvc, targetsSvc, nutSvc, stepsSvc, activityAI)
	webServer := tgapp.NewServer(tgapp.Deps{
		Addr:      cfg.WebAddr,
		BotToken:  cfg.BotToken,
		TZ:        cfg.TZ,
		Plan:      planSvc,
		Workout:   workoutSvc,
		Targets:   targetsSvc,
		Nutrition: nutSvc,
		Steps:     stepsSvc,
		Profile:   profileSvc,
		Training:  trainingProfileSvc,
		Inputs:    trainingInputSvc,
		Programs:  trainingProgramSvc,
		Injuries:  injuryTypeSvc,
		Activity:  activityAI,
		PlanView:  planView,
		StatsView: statsView,
	})
	go func() {
		if err := webServer.ListenAndServe(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("miniapp server error: %v", err)
		}
	}()

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

func runDay(ctx context.Context, api *tgbotapi.BotAPI, users *service.BotUsersService, nut *service.NutritionService, tz string) error {
	loc := util.MustLocation(tz)
	now := util.NowIn(loc)
	dayStart := util.DayStart(now, loc)
	dayEnd := dayStart.Add(24 * time.Hour)

	chatIDs, err := users.ListEnabled(ctx)
	if err != nil {
		return err
	}

	sent := 0
	for _, chatID := range chatIDs {
		kcal, _, _, _, err := nut.SumByDay(ctx, chatID, dayStart, dayEnd)
		if err != nil || kcal > 0 {
			continue
		}
		msg := tgbotapi.NewMessage(chatID, "Напоминание: добавь еду за сегодня (/setmeal)")
		if _, err := api.Send(msg); err != nil {
			continue
		}
		time.Sleep(60 * time.Millisecond)
		sent++
	}

	log.Printf("day: sent=%d", sent)
	return nil
}

func runEvening(ctx context.Context, api *tgbotapi.BotAPI, users *service.BotUsersService, nut *service.NutritionService, targets *service.TargetsService, workout *service.WorkoutService, steps *service.StepsService, tz string) error {
	loc := util.MustLocation(tz)
	now := util.NowIn(loc)
	dayStart := util.DayStart(now, loc)
	dayEnd := dayStart.Add(24 * time.Hour)
	dayDate := util.LocalDateStr(now, loc)

	chatIDs, err := users.ListEnabled(ctx)
	if err != nil {
		return err
	}

	sent := 0
	for _, chatID := range chatIDs {
		kcal, p, _, _, err := nut.SumByDay(ctx, chatID, dayStart, dayEnd)
		if err != nil {
			continue
		}

		proteinTarget := 170
		if tg, ok, _ := targets.Get(ctx, chatID); ok {
			proteinTarget = tg.ProteinG
		}

		if p < proteinTarget {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Белок недобор: %d / %d. Добавь еду (/setmeal).", p, proteinTarget))
			if _, err := api.Send(msg); err != nil {
				continue
			}
			time.Sleep(60 * time.Millisecond)
			sent++
		}

		if _, hasSteps, _ := steps.GetByDate(ctx, chatID, dayDate); !hasSteps {
			msg := tgbotapi.NewMessage(chatID, "Сколько шагов сегодня? Ответь командой /setstep 12345")
			if _, err := api.Send(msg); err == nil {
				time.Sleep(60 * time.Millisecond)
				sent++
			}
		}

		hard, err := users.GetHard(ctx, chatID)
		if err != nil || !hard {
			continue
		}

		status, hasWorkout, _ := workout.GetStatusByDate(ctx, chatID, dayDate)
		if hasWorkout && status == "skip" {
			msg := tgbotapi.NewMessage(chatID, "Тренировка пропущена. Решай.")
			if _, err := api.Send(msg); err == nil {
				time.Sleep(60 * time.Millisecond)
			}
		}

		if kcal == 0 {
			yesterday := dayStart.AddDate(0, 0, -1)
			yKcal, _, _, _, _ := nut.SumByDay(ctx, chatID, yesterday, dayStart)
			if yKcal == 0 {
				msg := tgbotapi.NewMessage(chatID, "Сегодня пусто. Решай.")
				if _, err := api.Send(msg); err == nil {
					time.Sleep(60 * time.Millisecond)
				}
			}
		}
	}

	log.Printf("evening: sent=%d", sent)
	return nil
}

func runWeekly(ctx context.Context, api *tgbotapi.BotAPI, users *service.BotUsersService, nut *service.NutritionService, workout *service.WorkoutService, targets *service.TargetsService, ai *service.ReflectionAI, tz string) error {
	loc := util.MustLocation(tz)
	now := util.NowIn(loc)
	weekday := util.Weekday1to7(now)
	weekStart := util.DayStart(now.AddDate(0, 0, -(weekday-1)), loc)
	weekEnd := weekStart.AddDate(0, 0, 6)

	chatIDs, err := users.ListEnabled(ctx)
	if err != nil {
		return err
	}

	sent := 0
	for _, chatID := range chatIDs {
		foodMap, _ := nut.SumByRangeDaily(ctx, chatID, weekStart, weekEnd.Add(24*time.Hour), tz)
		workoutMap, _ := workout.ListByRange(ctx, chatID, util.LocalDateStr(weekStart, loc), util.LocalDateStr(weekEnd, loc))

		totalKcal := 0
		totalProtein := 0
		emptyDays := 0
		for i := 0; i < 7; i++ {
			d := weekStart.AddDate(0, 0, i)
			key := util.LocalDateStr(d, loc)
			if dn, ok := foodMap[key]; ok {
				totalKcal += dn.Kcal
				totalProtein += dn.P
				if dn.Kcal == 0 {
					emptyDays++
				}
			} else {
				emptyDays++
			}
		}

		done := 0
		totalWorkouts := 0
		for _, st := range workoutMap {
			if st == "done" || st == "skip" {
				totalWorkouts++
				if st == "done" {
					done++
				}
			}
		}

		avgKcal := totalKcal / 7
		avgProtein := totalProtein / 7
		proteinTarget := 170
		if tg, ok, _ := targets.Get(ctx, chatID); ok {
			proteinTarget = tg.ProteinG
		}

		mainIssue := "стабильно"
		if totalWorkouts > 0 && done < totalWorkouts {
			mainIssue = "пропуски тренировок"
		} else if avgProtein < proteinTarget {
			mainIssue = "низкий белок"
		} else if emptyDays > 0 {
			mainIssue = "пропуски еды"
		}

		text, err := ai.WeeklyReflection(ctx, done, totalWorkouts, avgKcal, avgProtein, proteinTarget, mainIssue)
		if err != nil {
			text = fmt.Sprintf("Неделя: Тренировок %d/%d. Средние калории %d. Белок %d. Главный косяк: %s.", done, totalWorkouts, avgKcal, avgProtein, mainIssue)
		}

		msg := tgbotapi.NewMessage(chatID, text)
		if _, err := api.Send(msg); err != nil {
			continue
		}
		time.Sleep(60 * time.Millisecond)
		sent++
	}

	log.Printf("weekly: sent=%d", sent)
	return nil
}

func runWeightReminder(ctx context.Context, api *tgbotapi.BotAPI, users *service.BotUsersService, profiles *service.ProfileService, tz string) error {
	loc := util.MustLocation(tz)
	now := util.NowIn(loc)
	if util.Weekday1to7(now) != 1 {
		log.Printf("weight: skipped (not monday)")
		return nil
	}

	chatIDs, err := users.ListEnabled(ctx)
	if err != nil {
		return err
	}

	sent := 0
	for _, chatID := range chatIDs {
		if _, ok, err := profiles.Get(ctx, chatID); err != nil || !ok {
			continue
		}
		msg := tgbotapi.NewMessage(chatID, "Понедельник — обнови вес. Напиши /weight 82.5")
		if _, err := api.Send(msg); err != nil {
			continue
		}
		time.Sleep(60 * time.Millisecond)
		sent++
	}

	log.Printf("weight: sent=%d", sent)
	return nil
}
