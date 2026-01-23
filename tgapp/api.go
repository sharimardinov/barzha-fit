package tgapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"barzhafit/backend/domain"
	"barzhafit/backend/service"
	"barzhafit/backend/storage/db"
	"barzhafit/backend/util"
)

type apiResponse struct {
	OK    bool        `json:"ok"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

func (s *Server) registerAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/today", s.withAuth(s.handleToday))
	mux.HandleFunc("/api/workout/set", s.withAuth(s.handleWorkoutSet))
	mux.HandleFunc("/api/meals/today", s.withAuth(s.handleMealsToday))
	mux.HandleFunc("/api/meals/recent", s.withAuth(s.handleMealsRecent))
	mux.HandleFunc("/api/meal/add", s.withAuth(s.handleMealAdd))
	mux.HandleFunc("/api/meal/delete", s.withAuth(s.handleMealDelete))
	mux.HandleFunc("/api/meal/undo", s.withAuth(s.handleMealUndo))
	mux.HandleFunc("/api/steps/set", s.withAuth(s.handleStepsSet))
	mux.HandleFunc("/api/targets/get", s.withAuth(s.handleTargetsGet))
	mux.HandleFunc("/api/targets/set", s.withAuth(s.handleTargetsSet))
	mux.HandleFunc("/api/targets/refresh", s.withAuth(s.handleTargetsRefresh))
	mux.HandleFunc("/api/plan/get", s.withAuth(s.handlePlanGet))
	mux.HandleFunc("/api/plan/set", s.withAuth(s.handlePlanSet))
	mux.HandleFunc("/api/profile/get", s.withAuth(s.handleProfileGet))
	mux.HandleFunc("/api/profile/set", s.withAuth(s.handleProfileSet))
	mux.HandleFunc("/api/activity/estimate", s.withAuth(s.handleActivityEstimate))
	mux.HandleFunc("/api/training/profile/get", s.withAuth(s.handleTrainingProfileGet))
	mux.HandleFunc("/api/training/profile/set", s.withAuth(s.handleTrainingProfileSet))
	mux.HandleFunc("/api/training/injuries", s.withAuth(s.handleTrainingInjuries))
	mux.HandleFunc("/api/training/program/generate", s.withAuth(s.handleTrainingProgramGenerate))
	mux.HandleFunc("/api/weight/set", s.withAuth(s.handleWeightSet))
	mux.HandleFunc("/api/stats/week", s.withAuth(s.handleStatsWeek))
	mux.HandleFunc("/api/stats/month", s.withAuth(s.handleStatsMonth))
	mux.HandleFunc("/api/stats/strength", s.withAuth(s.handleStrengthStats))
	mux.HandleFunc("/api/streak", s.withAuth(s.handleStreak))
	mux.HandleFunc("/api/workout/plan/get", s.withAuth(s.handleWorkoutPlanGet))
	mux.HandleFunc("/api/workout/plan/save", s.withAuth(s.handleWorkoutPlanSave))
	mux.HandleFunc("/api/workout/session/get", s.withAuth(s.handleWorkoutSessionGet))
	mux.HandleFunc("/api/workout/session/start", s.withAuth(s.handleWorkoutSessionStart))
	mux.HandleFunc("/api/workout/session/warmup/end", s.withAuth(s.handleWorkoutWarmupEnd))
	mux.HandleFunc("/api/workout/session/rest/end", s.withAuth(s.handleWorkoutRestEnd))
	mux.HandleFunc("/api/workout/session/set/finish", s.withAuth(s.handleWorkoutSetFinish))
	mux.HandleFunc("/api/workout/session/pause", s.withAuth(s.handleWorkoutSessionPause))
	mux.HandleFunc("/api/workout/session/resume", s.withAuth(s.handleWorkoutSessionResume))
	mux.HandleFunc("/api/workout/session/stop", s.withAuth(s.handleWorkoutSessionStop))
}

func (s *Server) withAuth(next func(http.ResponseWriter, *http.Request, authContext)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, apiResponse{OK: false, Error: "method_not_allowed"})
			return
		}
		auth, err := s.authenticate(r)
		if err != nil {
			errCode := "unauthorized"
			switch {
			case errors.Is(err, errMissingAuth):
				errCode = "missing_auth"
			case errors.Is(err, errMissingInit):
				errCode = "missing_init_data"
			case errors.Is(err, errBadHash):
				errCode = "bad_hash"
			case errors.Is(err, errStaleAuthDate):
				errCode = "stale_auth_date"
			case errors.Is(err, errBadInitData):
				errCode = "bad_init_data"
			case errors.Is(err, errBadUserPayload):
				errCode = "bad_user_payload"
			}
			writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: errCode})
			return
		}
		next(w, r, auth)
	}
}

func (s *Server) handleToday(w http.ResponseWriter, r *http.Request, auth authContext) {
	ctx := context.Background()
	chatID := auth.User.ID
	loc := util.MustLocation(s.tz)
	now := util.NowIn(loc)
	dayDate := util.LocalDateStr(now, loc)
	cycleDay := util.Weekday1to7(now)

	planText, err := s.plan.Get(ctx, chatID)
	block := "Нет плана. Сначала /setplan"
	if err == nil {
		days := service.SplitPlanByDays(planText)
		block = strings.TrimSpace(days[cycleDay])
		if block == "" {
			block = "День не найден в плане. Проверь заголовки 1..7 отдельной строкой."
		}
	}

	status, hasWorkout, _ := s.workout.GetStatusByDate(ctx, chatID, dayDate)
	workoutIcon := "—"
	if hasWorkout {
		if status == "done" {
			workoutIcon = "✅"
		} else if status == "skip" {
			workoutIcon = "❌"
		}
	}

	kcal, p, f, c, _ := s.nutrition.SumToday(ctx, chatID, loc, now)
	steps, hasSteps, _ := s.steps.GetByDate(ctx, chatID, dayDate)
	if !hasSteps {
		steps = 0
	}

	kcalTarget := 2400
	proteinTarget := 170
	fatTarget := 70
	carbsTarget := 250
	stepsTarget := 10000
	if tg, ok, _ := s.targets.Get(ctx, chatID); ok {
		kcalTarget = tg.Kcal
		proteinTarget = tg.ProteinG
		fatTarget = tg.FatG
		carbsTarget = tg.CarbsG
		if tg.Steps > 0 {
			stepsTarget = tg.Steps
		}
	}

	resp := map[string]interface{}{
		"date":        dayDate,
		"cycleDay":    cycleDay,
		"plan":        block,
		"workout":     status,
		"workoutIcon": workoutIcon,
		"kcal":        kcal,
		"protein":     p,
		"fat":         f,
		"carbs":       c,
		"steps":       steps,
		"targets": map[string]int{
			"kcal":    kcalTarget,
			"protein": proteinTarget,
			"fat":     fatTarget,
			"carbs":   carbsTarget,
			"steps":   stepsTarget,
		},
		"icons": map[string]string{
			"kcal":    ratioIcon(float64(kcal), float64(kcalTarget)),
			"protein": proteinRatioIcon(float64(p), float64(proteinTarget)),
			"fat":     ratioIcon(float64(f), float64(fatTarget)),
			"carbs":   ratioIcon(float64(c), float64(carbsTarget)),
			"steps":   ratioIcon(float64(steps), float64(stepsTarget)),
			"food":    ratioIcon(float64(kcal), float64(kcalTarget)),
		},
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: resp})
}

func (s *Server) handleWorkoutSet(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_request"})
		return
	}
	if payload.Status != "done" && payload.Status != "skip" {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid_status"})
		return
	}
	loc := util.MustLocation(s.tz)
	now := util.NowIn(loc)
	dayDate := util.LocalDateStr(now, loc)
	if _, err := s.workout.MarkAndAdvance(context.Background(), auth.User.ID, dayDate, payload.Status); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "save_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (s *Server) handleMealsToday(w http.ResponseWriter, r *http.Request, auth authContext) {
	loc := util.MustLocation(s.tz)
	now := util.NowIn(loc)
	items, err := s.nutrition.ListToday(context.Background(), auth.User.ID, loc, now)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "meals_read_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: mealsToDTO(items)})
}

func (s *Server) handleMealsRecent(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		Limit int `json:"limit"`
	}
	_ = decodeJSON(r, &payload)
	if payload.Limit <= 0 || payload.Limit > 20 {
		payload.Limit = 10
	}
	items, err := s.nutrition.ListRecent(context.Background(), auth.User.ID, payload.Limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "meals_read_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: mealsToDTO(items)})
}

func (s *Server) handleMealAdd(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		Text string `json:"text"`
	}
	if err := decodeJSON(r, &payload); err != nil || strings.TrimSpace(payload.Text) == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "empty_text"})
		return
	}
	loc := util.MustLocation(s.tz)
	now := util.NowIn(loc)
	meal, err := s.nutrition.AddMealFromText(context.Background(), auth.User.ID, now, payload.Text)
	if err != nil && !errors.Is(err, service.ErrNutritionAI) {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "meal_save_failed"})
		return
	}
	resp := map[string]interface{}{
		"meal":    mealToDTO(meal),
		"aiError": errors.Is(err, service.ErrNutritionAI),
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: resp})
}

func (s *Server) handleMealDelete(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		ID int64 `json:"id"`
	}
	if err := decodeJSON(r, &payload); err != nil || payload.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_request"})
		return
	}
	ok, err := s.nutrition.DeleteByID(context.Background(), auth.User.ID, payload.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "meal_delete_failed"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, apiResponse{OK: false, Error: "not_found"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (s *Server) handleMealUndo(w http.ResponseWriter, r *http.Request, auth authContext) {
	ok, err := s.nutrition.UndoLast(context.Background(), auth.User.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "meal_delete_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]bool{"deleted": ok}})
}

func (s *Server) handleStepsSet(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		Steps int `json:"steps"`
	}
	if err := decodeJSON(r, &payload); err != nil || payload.Steps < 0 {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_steps"})
		return
	}
	loc := util.MustLocation(s.tz)
	now := util.NowIn(loc)
	dayDate := util.LocalDateStr(now, loc)
	if err := s.steps.SetSteps(context.Background(), auth.User.ID, dayDate, payload.Steps); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "steps_save_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (s *Server) handleTargetsGet(w http.ResponseWriter, r *http.Request, auth authContext) {
	kcalTarget := 2400
	proteinTarget := 170
	fatTarget := 70
	carbsTarget := 250
	stepsTarget := 10000
	source := "default"
	if tg, ok, _ := s.targets.Get(context.Background(), auth.User.ID); ok {
		kcalTarget = tg.Kcal
		proteinTarget = tg.ProteinG
		fatTarget = tg.FatG
		carbsTarget = tg.CarbsG
		if tg.Steps > 0 {
			stepsTarget = tg.Steps
		}
		source = tg.Source
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"kcal":    kcalTarget,
		"protein": proteinTarget,
		"fat":     fatTarget,
		"carbs":   carbsTarget,
		"steps":   stepsTarget,
		"source":  source,
	}})
}

func (s *Server) handleTargetsSet(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		Field string `json:"field"`
		Value int    `json:"value"`
	}
	if err := decodeJSON(r, &payload); err != nil || payload.Value < 0 {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_request"})
		return
	}
	if err := s.targets.SetManual(context.Background(), auth.User.ID, payload.Field, payload.Value); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "targets_update_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (s *Server) handleTargetsRefresh(w http.ResponseWriter, r *http.Request, auth authContext) {
	t, err := s.targets.Refresh(context.Background(), auth.User.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "targets_refresh_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: t})
}

func (s *Server) handlePlanGet(w http.ResponseWriter, r *http.Request, auth authContext) {
	planText, err := s.plan.Get(context.Background(), auth.User.ID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, apiResponse{OK: false, Error: "plan_not_found"})
		return
	}
	if normalized, ok := service.NormalizeTrainingPlan(planText); ok {
		planText = normalizeTrainingPlanTypes(normalized)
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"text": planText}})
}

func (s *Server) handlePlanSet(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		Text string `json:"text"`
	}
	if err := decodeJSON(r, &payload); err != nil || strings.TrimSpace(payload.Text) == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "empty_text"})
		return
	}
	if normalized, ok := service.NormalizeTrainingPlan(payload.Text); ok {
		payload.Text = normalizeTrainingPlanTypes(normalized)
		if tp, ok := service.ParseTrainingPlan(payload.Text); ok {
			issues := make([]string, 0)
			if !hasWeekPlanPayload(payload.Text) {
				issues = validateTrainingPlan(tp)
			}
			if payloadIssues := validateTrainingPlanPayload(payload.Text); len(payloadIssues) > 0 {
				issues = append(issues, payloadIssues...)
			}
			if len(issues) > 0 {
				writeJSON(w, http.StatusUnprocessableEntity, apiResponse{
					OK:    false,
					Error: "training_plan_invalid",
					Data:  map[string]any{"issues": issues},
				})
				return
			}
		}
	}
	if err := s.plan.Save(context.Background(), auth.User.ID, payload.Text); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "plan_save_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (s *Server) handleProfileGet(w http.ResponseWriter, r *http.Request, auth authContext) {
	p, ok, err := s.profile.Get(context.Background(), auth.User.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "profile_read_failed"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, apiResponse{OK: false, Error: "profile_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: profileToDTO(p)})
}

func (s *Server) handleProfileSet(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		Sex           string  `json:"sex"`
		Age           int     `json:"age"`
		HeightCM      int     `json:"height_cm"`
		WeightKG      float64 `json:"weight_kg"`
		BodyFat       float64 `json:"bodyfat_pct"`
		Activity      string  `json:"activity"`
		Goal          string  `json:"goal"`
		TrainingYears int     `json:"training_years"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_request"})
		return
	}

	p, ok, _ := s.profile.Get(context.Background(), auth.User.ID)
	if !ok {
		p = domain.Profile{ChatID: auth.User.ID, Activity: "mid", Goal: "balance"}
	}
	if payload.Sex != "" {
		p.Sex = payload.Sex
	}
	if payload.Age > 0 {
		p.Age = payload.Age
	}
	if payload.HeightCM > 0 {
		p.HeightCM = payload.HeightCM
	}
	if payload.WeightKG > 0 {
		p.WeightKG = payload.WeightKG
	}
	if payload.BodyFat > 0 {
		p.BodyFatPct = payload.BodyFat
	}
	if payload.Activity != "" {
		p.Activity = payload.Activity
	}
	if payload.Goal != "" {
		p.Goal = payload.Goal
	}
	if payload.TrainingYears >= 0 {
		p.TrainingYears = payload.TrainingYears
	}

	if err := s.profile.Save(context.Background(), p); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "profile_save_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: profileToDTO(p)})
}

func (s *Server) handleTrainingProfileGet(w http.ResponseWriter, r *http.Request, auth authContext) {
	if s.training == nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "training_profile_unavailable"})
		return
	}
	p, ok, err := s.training.Get(context.Background(), auth.User.ID)
	if err != nil {
		log.Printf("training profile get failed: chat_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "training_profile_read_failed"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: trainingProfileDTO{}})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: trainingProfileToDTO(p)})
}

func (s *Server) handleTrainingProfileSet(w http.ResponseWriter, r *http.Request, auth authContext) {
	if s.training == nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "training_profile_unavailable"})
		return
	}
	var payload struct {
		BenchKG          int     `json:"bench_kg"`
		Pullups          int     `json:"pullups"`
		RunKM            float64 `json:"run_km"`
		Injuries         string  `json:"injuries"`
		Goal             string  `json:"goal"`
		Pharma           *bool   `json:"pharma"`
		TrainingsPerWeek int     `json:"trainings_per_week"`
		Wishes           string  `json:"wishes"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_request"})
		return
	}

	p := domain.TrainingProfile{
		ChatID:           auth.User.ID,
		BenchKG:          payload.BenchKG,
		Pullups:          payload.Pullups,
		RunKM:            payload.RunKM,
		Injuries:         trimLimit(payload.Injuries, 400),
		Goal:             trimLimit(payload.Goal, 200),
		Pharma:           payload.Pharma,
		TrainingsPerWeek: payload.TrainingsPerWeek,
		Wishes:           trimLimit(payload.Wishes, 200),
	}

	if err := s.training.Save(context.Background(), p); err != nil {
		log.Printf("training profile save failed: chat_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "training_profile_save_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: trainingProfileToDTO(p)})
}

func (s *Server) handleTrainingInjuries(w http.ResponseWriter, r *http.Request, _ authContext) {
	if s.injuries == nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "injury_types_unavailable"})
		return
	}
	items, err := s.injuries.List(context.Background())
	if err != nil {
		log.Printf("injury types list failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "injury_types_read_failed"})
		return
	}
	type injuryDTO struct {
		Code  string `json:"code"`
		Label string `json:"label"`
	}
	resp := make([]injuryDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, injuryDTO{Code: item.Code, Label: item.Label})
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: resp})
}

func (s *Server) handleTrainingProgramGenerate(w http.ResponseWriter, r *http.Request, auth authContext) {
	if s.inputs == nil || s.programs == nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "training_program_unavailable"})
		return
	}

	var payload struct {
		FitnessLevel string   `json:"fitness_level"`
		Goal         string   `json:"goal"`
		DaysPerWeek  int      `json:"days_per_week"`
		Injuries     []string `json:"injuries"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_request"})
		return
	}

	ctx := context.Background()
	if _, err := s.inputs.SaveFromSelection(ctx, auth.User.ID, payload.FitnessLevel, payload.Goal, payload.DaysPerWeek, payload.Injuries); err != nil {
		log.Printf("training input save failed: chat_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "training_input_invalid"})
		return
	}

	_, program, err := s.programs.Generate(ctx, auth.User.ID)
	if err != nil {
		log.Printf("training program generate failed: chat_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "training_program_generate_failed"})
		return
	}

	planPayload, err := buildTrainingPlanPayload(program, payload.DaysPerWeek)
	if err != nil {
		log.Printf("training plan build failed: chat_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "training_plan_build_failed"})
		return
	}
	planTextRaw, err := json.Marshal(planPayload)
	if err != nil {
		log.Printf("training plan marshal failed: chat_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "training_plan_build_failed"})
		return
	}
	planText := string(planTextRaw)
	text := service.FormatGeneratedProgram(program)
	if s.plan != nil {
		if err := s.plan.Save(ctx, auth.User.ID, planText); err != nil {
			log.Printf("training program plan save failed: chat_id=%d err=%v", auth.User.ID, err)
			writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "plan_save_failed"})
			return
		}
	}

	resp := struct {
		Program domain.GeneratedProgram `json:"program"`
		Text    string                  `json:"text"`
	}{
		Program: program,
		Text:    text,
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: resp})
}

func buildTrainingPlanPayload(program domain.GeneratedProgram, daysPerWeek int) (trainingPlanPayload, error) {
	slots, err := trainingDaySlots(daysPerWeek)
	if err != nil {
		return trainingPlanPayload{}, err
	}
	if len(program.Days) != len(slots) {
		return trainingPlanPayload{}, fmt.Errorf("plan_days_mismatch")
	}
	slotSet := make(map[int]bool, len(slots))
	for _, day := range slots {
		slotSet[day] = true
	}

	week := make([]trainingPlanDay, 0, 7)
	programIndex := 0
	for dayNum := 1; dayNum <= 7; dayNum++ {
		if !slotSet[dayNum] {
			week = append(week, trainingPlanDay{
				Day:   dayNum,
				Name:  "Отдых",
				Focus: "",
				Type:  "rest",
				Items: []string{"Отдых"},
			})
			continue
		}

		day := program.Days[programIndex]
		programIndex++
		items := make([]string, 0, len(day.Exercises))
		for _, ex := range day.Exercises {
			items = append(items, formatGeneratedExerciseLine(ex))
		}
		name := strings.TrimSpace(day.Name)
		if name == "" {
			name = fmt.Sprintf("День %d", dayNum)
		}
		week = append(week, trainingPlanDay{
			Day:   dayNum,
			Name:  name,
			Focus: strings.TrimSpace(day.Focus),
			Type:  "train",
			Items: items,
		})
	}

	return trainingPlanPayload{Week: week}, nil
}

func trainingDaySlots(daysPerWeek int) ([]int, error) {
	switch daysPerWeek {
	case 2:
		return []int{1, 4}, nil
	case 3:
		return []int{1, 3, 5}, nil
	case 4:
		return []int{1, 2, 4, 5}, nil
	case 5:
		return []int{1, 2, 4, 5, 7}, nil
	case 6:
		return []int{1, 2, 3, 5, 6, 7}, nil
	default:
		return nil, fmt.Errorf("unsupported days_per_week")
	}
}

func formatGeneratedExerciseLine(ex domain.GeneratedExercise) string {
	line := fmt.Sprintf("%s — %dx%s", ex.Name, ex.Sets, ex.Reps)
	extras := make([]string, 0, 4)
	if strings.TrimSpace(ex.RPE) != "" {
		extras = append(extras, "RPE "+strings.TrimSpace(ex.RPE))
	}
	if strings.TrimSpace(ex.Rest) != "" {
		extras = append(extras, "Rest "+strings.TrimSpace(ex.Rest))
	}
	if strings.TrimSpace(ex.Percent1RM) != "" {
		extras = append(extras, "%1RM "+strings.TrimSpace(ex.Percent1RM))
	}
	if tagLine := formatTagLine(ex.Tags); tagLine != "" {
		extras = append(extras, tagLine)
	}
	if len(extras) > 0 {
		line += " | " + strings.Join(extras, " | ")
	}
	return line
}

func formatTagLine(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		t := strings.TrimSpace(tag)
		if t == "" {
			continue
		}
		if !strings.HasPrefix(t, "#") {
			t = "#" + t
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, " ")
}

func (s *Server) handleWeightSet(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		WeightKG float64 `json:"weight_kg"`
	}
	if err := decodeJSON(r, &payload); err != nil || payload.WeightKG <= 0 {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_weight"})
		return
	}
	p, ok, err := s.profile.UpdateWeight(context.Background(), auth.User.ID, payload.WeightKG)
	if err != nil || !ok {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "weight_update_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: profileToDTO(p)})
}

func (s *Server) handleActivityEstimate(w http.ResponseWriter, r *http.Request, auth authContext) {
	if s.activity == nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "activity_ai_unavailable"})
		return
	}
	ctx := context.Background()
	p, ok, err := s.profile.Get(ctx, auth.User.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "profile_read_failed"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "profile_not_found"})
		return
	}
	planText, err := s.plan.Get(ctx, auth.User.ID)
	if err != nil || strings.TrimSpace(planText) == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "plan_not_found"})
		return
	}
	mult, raw, err := s.activity.EstimateActivityMultiplierWithProfile(ctx, planText, p)
	if err != nil {
		log.Printf("activity estimate failed: chat_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{
			OK:    false,
			Error: "activity_estimate_failed",
			Data:  raw,
		})
		return
	}
	p.Activity = fmt.Sprintf("ai:%.2f", mult)
	if err := s.profile.Save(ctx, p); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "profile_save_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: profileToDTO(p)})
}

func (s *Server) handleStatsWeek(w http.ResponseWriter, r *http.Request, auth authContext) {
	ctx := context.Background()
	loc := util.MustLocation(s.tz)
	now := util.NowIn(loc)
	weekday := util.Weekday1to7(now)
	weekStart := util.DayStart(now.AddDate(0, 0, -(weekday-1)), loc)
	weekDays := weekday
	weekEnd := weekStart.AddDate(0, 0, weekDays-1)

	foodMap, _ := s.nutrition.SumByRangeDaily(ctx, auth.User.ID, weekStart, weekEnd.Add(24*time.Hour), s.tz)
	stepsMap, _ := s.steps.ListByRange(ctx, auth.User.ID, util.LocalDateStr(weekStart, loc), util.LocalDateStr(weekEnd, loc))

	kcalTarget := 2400
	stepsTarget := 10000
	if tg, ok, _ := s.targets.Get(ctx, auth.User.ID); ok {
		kcalTarget = tg.Kcal
		if tg.Steps > 0 {
			stepsTarget = tg.Steps
		}
	}

	days := make([]map[string]interface{}, 0, weekDays)
	for i := 0; i < weekDays; i++ {
		dayStart := util.DayStart(weekStart.AddDate(0, 0, i), loc)
		dayDate := util.LocalDateStr(dayStart, loc)
		kcal := 0
		if dn, ok := foodMap[dayDate]; ok {
			kcal = dn.Kcal
		}
		steps := 0
		if v, ok := stepsMap[dayDate]; ok {
			steps = v
		}
		days = append(days, map[string]interface{}{
			"day":     dayStart.Day(),
			"date":    dayDate,
			"foodOk":  foodInRange(kcal, kcalTarget),
			"stepsOk": steps >= stepsTarget,
		})
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"days": days,
	}})
}

func (s *Server) handleStatsMonth(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		Offset int `json:"offset"`
	}
	_ = decodeJSON(r, &payload)
	if payload.Offset > 0 {
		payload.Offset = 0
	}

	ctx := context.Background()
	loc := util.MustLocation(s.tz)
	now := util.NowIn(loc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, payload.Offset, 0)
	monthDays := daysInMonth(monthStart)
	monthEnd := monthStart.AddDate(0, 0, monthDays-1)
	offset := util.Weekday1to7(monthStart) - 1

	foodMap, _ := s.nutrition.SumByRangeDaily(ctx, auth.User.ID, monthStart, monthEnd.Add(24*time.Hour), s.tz)
	stepsMap, _ := s.steps.ListByRange(ctx, auth.User.ID, util.LocalDateStr(monthStart, loc), util.LocalDateStr(monthEnd, loc))

	kcalTarget := 2400
	stepsTarget := 10000
	if tg, ok, _ := s.targets.Get(ctx, auth.User.ID); ok {
		kcalTarget = tg.Kcal
		if tg.Steps > 0 {
			stepsTarget = tg.Steps
		}
	}

	days := make([]map[string]interface{}, 0, monthDays)
	for i := 0; i < monthDays; i++ {
		dayStart := monthStart.AddDate(0, 0, i)
		dayDate := util.LocalDateStr(dayStart, loc)
		kcal := 0
		if dn, ok := foodMap[dayDate]; ok {
			kcal = dn.Kcal
		}
		steps := 0
		if v, ok := stepsMap[dayDate]; ok {
			steps = v
		}
		days = append(days, map[string]interface{}{
			"day":     dayStart.Day(),
			"date":    dayDate,
			"foodOk":  foodInRange(kcal, kcalTarget),
			"stepsOk": steps >= stepsTarget,
		})
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"offset":     offset,
		"days":       days,
		"monthStart": monthStart.Format("2006-01-02"),
	}})
}

func (s *Server) handleStrengthStats(w http.ResponseWriter, r *http.Request, auth authContext) {
	if s.strengthStats == nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "strength_stats_unavailable"})
		return
	}
	ctx := context.Background()
	stats, err := s.strengthStats.StrengthAllTime(ctx, auth.User.ID)
	if err != nil {
		log.Printf("strength stats failed: chat_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "strength_stats_failed"})
		return
	}
	stepsTotal := 0
	if s.steps != nil {
		stepsTotal, err = s.steps.SumAllTime(ctx, auth.User.ID)
		if err != nil {
			log.Printf("steps stats failed: chat_id=%d err=%v", auth.User.ID, err)
			writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "strength_stats_failed"})
			return
		}
	}
	protein, fat, carbs := 0, 0, 0
	if s.nutrition != nil {
		_, protein, fat, carbs, err = s.nutrition.SumAllTime(ctx, auth.User.ID)
		if err != nil {
			log.Printf("nutrition stats failed: chat_id=%d err=%v", auth.User.ID, err)
			writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "strength_stats_failed"})
			return
		}
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"strength":   stats,
		"stepsTotal": stepsTotal,
		"macros": map[string]int{
			"protein": protein,
			"fat":     fat,
			"carbs":   carbs,
		},
	}})
}

func (s *Server) handleStreak(w http.ResponseWriter, r *http.Request, auth authContext) {
	loc := util.MustLocation(s.tz)
	now := util.NowIn(loc)
	from := util.DayStart(now.AddDate(0, 0, -60), loc)
	to := util.DayStart(now, loc)

	workoutMap, _ := s.workout.ListByRange(context.Background(), auth.User.ID, util.LocalDateStr(from, loc), util.LocalDateStr(to, loc))
	mealMap, _ := s.nutrition.SumByRangeDaily(context.Background(), auth.User.ID, from, to.Add(24*time.Hour), s.tz)

	workoutStreak := 0
	for i := 0; i <= 60; i++ {
		d := util.DayStart(now.AddDate(0, 0, -i), loc)
		key := util.LocalDateStr(d, loc)
		status, ok := workoutMap[key]
		if !ok || status != "done" {
			break
		}
		workoutStreak++
	}

	noMealStreak := 0
	for i := 0; i <= 60; i++ {
		d := util.DayStart(now.AddDate(0, 0, -i), loc)
		key := util.LocalDateStr(d, loc)
		kcal := 0
		if dn, ok := mealMap[key]; ok {
			kcal = dn.Kcal
		}
		if kcal > 0 {
			break
		}
		noMealStreak++
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"workoutStreak": workoutStreak,
		"mealStreak":    noMealStreak,
		"mealBar":       streakBar(noMealStreak, 7),
	}})
}

func (s *Server) handleWorkoutPlanGet(w http.ResponseWriter, r *http.Request, auth authContext) {
	plan, err := s.workoutPlanFromToday(context.Background(), auth.User.ID)
	if err != nil {
		if writeWorkoutError(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "workout_plan_read_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"plan": plan,
	}})
}

func (s *Server) handleWorkoutPlanSave(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		Plan domain.WorkoutPlan `json:"plan"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_request"})
		return
	}
	if _, err := s.workoutTimer.SavePlan(context.Background(), auth.User.ID, &payload.Plan); err != nil {
		if writeWorkoutError(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "workout_plan_save_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (s *Server) handleWorkoutSessionGet(w http.ResponseWriter, r *http.Request, auth authContext) {
	session, plan, err := s.workoutTimer.GetSession(context.Background(), auth.User.ID)
	if err != nil {
		if writeWorkoutError(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "workout_session_read_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"session": workoutSessionToDTO(session),
		"plan":    plan,
	}})
}

func (s *Server) handleWorkoutSessionStart(w http.ResponseWriter, r *http.Request, auth authContext) {
	plan, err := s.workoutPlanFromToday(context.Background(), auth.User.ID)
	if err != nil {
		if writeWorkoutError(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "workout_plan_read_failed"})
		return
	}
	session, existing, err := s.workoutTimer.StartSessionWithPlan(context.Background(), auth.User.ID, plan)
	if err != nil {
		if writeWorkoutError(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "workout_session_start_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"session":  workoutSessionToDTO(session),
		"plan":     plan,
		"existing": existing,
	}})
}

func (s *Server) handleWorkoutWarmupEnd(w http.ResponseWriter, r *http.Request, auth authContext) {
	session, plan, err := s.workoutTimer.FinishWarmup(context.Background(), auth.User.ID)
	if err != nil {
		if writeWorkoutError(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "workout_warmup_finish_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"session": workoutSessionToDTO(session),
		"plan":    plan,
	}})
}

func (s *Server) handleWorkoutRestEnd(w http.ResponseWriter, r *http.Request, auth authContext) {
	session, plan, err := s.workoutTimer.FinishRest(context.Background(), auth.User.ID)
	if err != nil {
		if writeWorkoutError(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "workout_rest_finish_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"session": workoutSessionToDTO(session),
		"plan":    plan,
	}})
}

func (s *Server) handleWorkoutSetFinish(w http.ResponseWriter, r *http.Request, auth authContext) {
	var payload struct {
		ExerciseIndex int     `json:"exerciseIndex"`
		SetIndex      int     `json:"setIndex"`
		ActualWeight  float64 `json:"actualWeight"`
		ActualReps    int     `json:"actualReps"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_request"})
		return
	}
	session, plan, err := s.workoutTimer.FinishSet(context.Background(), auth.User.ID, payload.ExerciseIndex, payload.SetIndex, payload.ActualWeight, payload.ActualReps)
	if err != nil {
		if writeWorkoutError(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "workout_set_finish_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"session": workoutSessionToDTO(session),
		"plan":    plan,
	}})
}

func (s *Server) handleWorkoutSessionPause(w http.ResponseWriter, r *http.Request, auth authContext) {
	session, plan, err := s.workoutTimer.Pause(context.Background(), auth.User.ID)
	if err != nil {
		if writeWorkoutError(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "workout_session_pause_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"session": workoutSessionToDTO(session),
		"plan":    plan,
	}})
}

func (s *Server) handleWorkoutSessionResume(w http.ResponseWriter, r *http.Request, auth authContext) {
	session, plan, err := s.workoutTimer.Resume(context.Background(), auth.User.ID)
	if err != nil {
		if writeWorkoutError(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "workout_session_resume_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"session": workoutSessionToDTO(session),
		"plan":    plan,
	}})
}

func (s *Server) handleWorkoutSessionStop(w http.ResponseWriter, r *http.Request, auth authContext) {
	session, plan, err := s.workoutTimer.StopSession(context.Background(), auth.User.ID)
	if err != nil {
		if writeWorkoutError(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "workout_session_stop_failed"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"session": workoutSessionToDTO(session),
		"plan":    plan,
	}})
}

func (s *Server) workoutPlanFromToday(ctx context.Context, chatID int64) (*domain.WorkoutPlan, error) {
	planText, err := s.plan.Get(ctx, chatID)
	if err != nil {
		return nil, service.ErrWorkoutPlanNotFound
	}
	loc := util.MustLocation(s.tz)
	now := util.NowIn(loc)
	day := util.Weekday1to7(now)
	days := service.SplitPlanByDays(planText)
	block := strings.TrimSpace(days[day])
	if block == "" {
		raw := strings.TrimSpace(planText)
		if raw != "" && !strings.HasPrefix(raw, "{") {
			block = raw
		}
	}
	if block == "" {
		return nil, service.WorkoutPlanValidationError{Issues: []string{"Сегодня нет тренировки"}}
	}
	plan, issues := service.BuildWorkoutPlanFromText(block)
	if len(issues) > 0 {
		return nil, service.WorkoutPlanValidationError{Issues: issues}
	}
	return &plan, nil
}

type workoutSessionDTO struct {
	ID               int64      `json:"id"`
	Status           string     `json:"status"`
	Phase            string     `json:"phase"`
	ExerciseIndex    int        `json:"exerciseIndex"`
	SetIndex         int        `json:"setIndex"`
	TimerKind        string     `json:"timerKind,omitempty"`
	TimerStartedAt   *time.Time `json:"timerStartedAt,omitempty"`
	TimerDurationSec int        `json:"timerDurationSec,omitempty"`
	WarmupEndedAt    *time.Time `json:"warmupEndedAt,omitempty"`
	PausedAt         *time.Time `json:"pausedAt,omitempty"`
	PausedTotalSec   int        `json:"pausedTotalSec"`
	StartedAt        time.Time  `json:"startedAt"`
}

func workoutSessionToDTO(s domain.WorkoutSession) workoutSessionDTO {
	return workoutSessionDTO{
		ID:               s.ID,
		Status:           s.Status,
		Phase:            s.Phase,
		ExerciseIndex:    s.ExerciseIndex,
		SetIndex:         s.SetIndex,
		TimerKind:        s.TimerKind,
		TimerStartedAt:   s.TimerStartedAt,
		TimerDurationSec: s.TimerDurationSec,
		WarmupEndedAt:    s.WarmupEndedAt,
		PausedAt:         s.PausedAt,
		PausedTotalSec:   s.PausedTotalSec,
		StartedAt:        s.StartedAt,
	}
}

func writeWorkoutError(w http.ResponseWriter, err error) bool {
	var ve service.WorkoutPlanValidationError
	switch {
	case errors.Is(err, service.ErrWorkoutPlanNotFound):
		writeJSON(w, http.StatusNotFound, apiResponse{OK: false, Error: "workout_plan_not_found"})
		return true
	case errors.Is(err, service.ErrWorkoutSessionNotFound):
		writeJSON(w, http.StatusNotFound, apiResponse{OK: false, Error: "workout_session_not_found"})
		return true
	case errors.Is(err, service.ErrWorkoutSessionState):
		writeJSON(w, http.StatusConflict, apiResponse{OK: false, Error: "workout_session_state"})
		return true
	case errors.Is(err, service.ErrWorkoutSessionPaused):
		writeJSON(w, http.StatusConflict, apiResponse{OK: false, Error: "workout_session_paused"})
		return true
	case errors.As(err, &ve):
		writeJSON(w, http.StatusUnprocessableEntity, apiResponse{OK: false, Error: "workout_plan_invalid", Data: map[string]interface{}{
			"issues": ve.Issues,
		}})
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, status int, resp apiResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func decodeJSON(r *http.Request, out interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func ratioIcon(val, target float64) string {
	if target <= 0 {
		return "—"
	}
	r := val / target
	if r >= 0.9 && r <= 1.1 {
		return "🟢"
	}
	return "🔴"
}

func proteinRatioIcon(val, target float64) string {
	if target <= 0 {
		return "—"
	}
	r := val / target
	if r >= 0.9 {
		return "🟢"
	}
	return "🔴"
}

func streakBar(val, max int) string {
	if max <= 0 {
		return ""
	}
	if val > max {
		val = max
	}
	var b strings.Builder
	b.WriteString("[")
	for i := 0; i < max; i++ {
		if i < val {
			b.WriteString("■")
		} else {
			b.WriteString("—")
		}
	}
	b.WriteString("]")
	return b.String()
}

func daysInMonth(start time.Time) int {
	return time.Date(start.Year(), start.Month()+1, 0, 0, 0, 0, 0, start.Location()).Day()
}

func foodInRange(kcal int, target int) bool {
	if kcal == 0 {
		return false
	}
	min := int(float64(target) * 0.9)
	max := int(float64(target) * 1.1)
	return kcal >= min && kcal <= max
}

type mealDTO struct {
	ID       int64  `json:"id"`
	EatenAt  string `json:"eaten_at"`
	Text     string `json:"text"`
	Kcal     int    `json:"kcal"`
	ProteinG int    `json:"protein_g"`
	FatG     int    `json:"fat_g"`
	CarbsG   int    `json:"carbs_g"`
}

type profileDTO struct {
	ChatID             int64   `json:"chat_id"`
	Sex                string  `json:"sex"`
	Age                int     `json:"age"`
	HeightCM           int     `json:"height_cm"`
	WeightKG           float64 `json:"weight_kg"`
	BodyFat            float64 `json:"bodyfat_pct"`
	Activity           string  `json:"activity"`
	ActivityMultiplier float64 `json:"activity_multiplier"`
	Goal               string  `json:"goal"`
	TrainingYears      int     `json:"training_years"`
}

type trainingProfileDTO struct {
	BenchKG          int     `json:"bench_kg"`
	Pullups          int     `json:"pullups"`
	RunKM            float64 `json:"run_km"`
	Injuries         string  `json:"injuries"`
	Goal             string  `json:"goal"`
	Pharma           *bool   `json:"pharma"`
	TrainingsPerWeek int     `json:"trainings_per_week"`
	Wishes           string  `json:"wishes"`
}

func profileToDTO(p domain.Profile) profileDTO {
	return profileDTO{
		ChatID:             p.ChatID,
		Sex:                p.Sex,
		Age:                p.Age,
		HeightCM:           p.HeightCM,
		WeightKG:           p.WeightKG,
		BodyFat:            p.BodyFatPct,
		Activity:           p.Activity,
		ActivityMultiplier: util.ActivityMultiplier(p.Activity),
		Goal:               p.Goal,
		TrainingYears:      p.TrainingYears,
	}
}

func trainingProfileToDTO(p domain.TrainingProfile) trainingProfileDTO {
	return trainingProfileDTO{
		BenchKG:          p.BenchKG,
		Pullups:          p.Pullups,
		RunKM:            p.RunKM,
		Injuries:         p.Injuries,
		Goal:             p.Goal,
		Pharma:           p.Pharma,
		TrainingsPerWeek: p.TrainingsPerWeek,
		Wishes:           p.Wishes,
	}
}

func missingTrainingFields(p domain.Profile, tp domain.TrainingProfile, hasProfile bool, hasTraining bool) []string {
	missing := make([]string, 0)
	if !hasProfile || p.Sex == "" {
		missing = append(missing, "пол")
	}
	if !hasProfile || p.Age < 14 || p.Age > 80 {
		missing = append(missing, "возраст")
	}
	if !hasProfile || p.HeightCM < 100 || p.HeightCM > 250 {
		missing = append(missing, "рост_см")
	}
	if !hasProfile || p.WeightKG < 30 || p.WeightKG > 300 {
		missing = append(missing, "вес_кг")
	}
	if !hasProfile || p.BodyFatPct < 1 || p.BodyFatPct > 100 {
		missing = append(missing, "процент_жира")
	}
	if !hasProfile || p.TrainingYears < 0 {
		missing = append(missing, "стаж_тренировок_лет")
	}
	if !hasTraining || tp.BenchKG < 0 || tp.BenchKG > 400 {
		missing = append(missing, "жим_лёжа_кг")
	}
	if !hasTraining || tp.Pullups < 0 || tp.Pullups > 100 {
		missing = append(missing, "подтягивания_раз")
	}
	if !hasTraining || tp.RunKM < 0 || tp.RunKM > 300 {
		missing = append(missing, "бег_км")
	}
	if !hasTraining || strings.TrimSpace(tp.Goal) == "" {
		missing = append(missing, "цель")
	}
	if !hasTraining || tp.Pharma == nil {
		missing = append(missing, "фармакология")
	}
	if !hasTraining || tp.TrainingsPerWeek < 1 || tp.TrainingsPerWeek > 7 {
		missing = append(missing, "тренировок_в_неделю")
	}
	return missing
}

func mealToDTO(m db.Meal) mealDTO {
	return mealDTO{
		ID:       m.ID,
		EatenAt:  m.EatenAt.Format(time.RFC3339),
		Text:     m.Text,
		Kcal:     m.ProteinG*4 + m.FatG*9 + m.CarbsG*4,
		ProteinG: m.ProteinG,
		FatG:     m.FatG,
		CarbsG:   m.CarbsG,
	}
}

func trimLimit(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) > max {
		return string(runes[:max])
	}
	return value
}

var exerciseLineRe = regexp.MustCompile(`(?i)\b\d+\s*[xх]\s*\d+`)

var allowedExerciseNames = map[string]struct{}{
	"жим штанги лежа (max.)":           {},
	"жим штанги лежа на накл.ск.":      {},
	"жим лежа на накл.ск. в тренажере": {},
	"жим гантелей лежа":                {},
	"жим гантелей лежа на накл.ск.":    {},
	"жим сидя": {},
	"разведение гантелей лежа": {},
	"бабочка": {},
	"сведение рук в кроссовере":              {},
	"тяга вертикального блока":               {},
	"тяга верт. блока обратный хват":         {},
	"тяга горизонтального блока/прям. ручка": {},
	"гребная тяга в тренажере (черн.)":       {},
	"гребная тяга в тренажере (зел.)":        {},
	"хаммер верхнего блока":                  {},
	"пулловер":                              {},
	"пулл эраунд":                           {},
	"тяга штанги в наклоне":                 {},
	"тяга гантелей в наклоне":               {},
	"шраги со штангой":                      {},
	"шраги с гантелями":                     {},
	"шраги в тренажере":                     {},
	"подтягивания в гравитроне":             {},
	"гиперэкстензия":                        {},
	"гиперэкстензия обратная":               {},
	"тяга т-образного грифа":                {},
	"тяга т-образного грифа в тренажере":    {},
	"тяга нижнего хаммера":                  {},
	"приседания со штангой":                 {},
	"приседания со штангой в смите":         {},
	"разгибание голени":                     {},
	"выпады с гантелями":                    {},
	"жим платформы ногами":                  {},
	"жим платформы паралельный":             {},
	"гакк приседания":                       {},
	"приседания с гантелями":                {},
	"сгибание голени":                       {},
	"сгибание голени сидя":                  {},
	"жим ногами высокая пост. стопы":        {},
	"сгибание голени в кроссовере":          {},
	"мертвая тяга":                          {},
	"приседание плие":                       {},
	"румынская тяга штанги":                 {},
	"румынская тяга на 1 ноге":              {},
	"ягодичный мостик":                      {},
	"болгарские приседания":                 {},
	"разгибание бедра в кроссовере":         {},
	"приведение бедра":                      {},
	"отведение бедра":                       {},
	"гиперэкстензия на ягодицы":             {},
	"жим гантелей сидя":                     {},
	"жим арнольда":                          {},
	"армейский жим стоя":                    {},
	"жим штанги сидя в смите":               {},
	"сгибание плеча в кроссовере":           {},
	"сгибание плеча с гантелями":            {},
	"обратный жим от плеч в тренажере":      {},
	"махи с гантелями стоя":                 {},
	"махи с гантелями сидя":                 {},
	"протяжка со штангой":                   {},
	"протяжка с гантелями":                  {},
	"махи в кроссовере (манжеты)":           {},
	"отведение плеча в кроссовере":          {},
	"отведение плеча в трен. бабочка":       {},
	"отведение плеча на скамье с гантел.":   {},
	"cгибание предплечья larry scott":       {},
	"сгибание предплечья larry scott":       {},
	"сгибание предплечья со штангой":        {},
	"сгибание предплечья с гантелями":       {},
	"сгибание предплечья в кроссовере":      {},
	"боковая тяга тросса на бицепс":         {},
	"фрунцузский жим с гантелями":           {},
	"фрунцузский жим со штангой":            {},
	"жим гантели из-за головы":              {},
	"разгибание предплечья в кроссовере":    {},
	"разгибание пред. из-за голов. в крос.": {},
	"жим лежа штанги узким хватом":          {},
	"поочередное разгибание пред. в крос.":  {},
}

var exerciseAliases = map[string]string{
	"жим штанги лежа":                            "жим штанги лежа (max.)",
	"жим штанги лежа на наклонной скамье":        "жим штанги лежа на накл.ск.",
	"жим штанги лежа на накл. ск.":               "жим штанги лежа на накл.ск.",
	"жим лежа на наклонной скамье в тренажере":   "жим лежа на накл.ск. в тренажере",
	"жим лежа на накл. ск. в тренажере":          "жим лежа на накл.ск. в тренажере",
	"жим гантелей лежа на наклонной скамье":      "жим гантелей лежа на накл.ск.",
	"жим гантелей лежа на накл. ск.":             "жим гантелей лежа на накл.ск.",
	"жим гантелей лежа под углом":                "жим гантелей лежа на накл.ск.",
	"жим гантелей под углом":                     "жим гантелей лежа на накл.ск.",
	"жим штанги сидя":                            "жим штанги сидя в смите",
	"тяга верхнего блока":                        "тяга вертикального блока",
	"тяга верхнего блока к груди":                "тяга вертикального блока",
	"тяга вертикального блока к груди":           "тяга вертикального блока",
	"тяга вертикального блока обратный хват":     "тяга верт. блока обратный хват",
	"тяга вертикального блока обратным хватом":   "тяга верт. блока обратный хват",
	"тяга горизонтального блока":                 "тяга горизонтального блока/прям. ручка",
	"тяга горизонтального блока прямая ручка":    "тяга горизонтального блока/прям. ручка",
	"тяга горизонтального блока с прямой ручкой": "тяга горизонтального блока/прям. ручка",
	"тяга гантели в наклоне":                     "тяга гантелей в наклоне",
	"гребная тяга в тренажере":                   "гребная тяга в тренажере (черн.)",
	"гребная тяга в тренажере черн.":             "гребная тяга в тренажере (черн.)",
	"пуловер":                                         "пулловер",
	"пуловер с гантелью":                              "пулловер",
	"пуловер с гантелями":                             "пулловер",
	"жим ногами":                                      "жим платформы ногами",
	"жим ногами в тренажере":                          "жим платформы ногами",
	"жим платформы":                                   "жим платформы ногами",
	"разгибание ног":                                  "разгибание голени",
	"разгибание ног в тренажере":                      "разгибание голени",
	"сгибание ног":                                    "сгибание голени",
	"сгибание ног сидя":                               "сгибание голени сидя",
	"сгибание ног в кроссовере":                       "сгибание голени в кроссовере",
	"присед со штангой":                               "приседания со штангой",
	"присед в смите":                                  "приседания со штангой в смите",
	"румынская тяга":                                  "румынская тяга штанги",
	"румынская тяга на одной ноге":                    "румынская тяга на 1 ноге",
	"ягодичный мост":                                  "ягодичный мостик",
	"сгибание плеча гантелями":                        "сгибание плеча с гантелями",
	"отведение плеча в тренажере бабочка":             "отведение плеча в трен. бабочка",
	"сгибание предплечья на скамье скотта":            "сгибание предплечья larry scott",
	"сгибание предплечья ларри скотт":                 "сгибание предплечья larry scott",
	"подъем гантелей на бицепс":                       "сгибание предплечья с гантелями",
	"подъем штанги на бицепс":                         "сгибание предплечья со штангой",
	"подъем на бицепс в кроссовере":                   "сгибание предплечья в кроссовере",
	"тяга троса на бицепс":                            "боковая тяга тросса на бицепс",
	"французский жим с гантелями":                     "фрунцузский жим с гантелями",
	"французский жим со штангой":                      "фрунцузский жим со штангой",
	"разгибание рук в кроссовере":                     "разгибание предплечья в кроссовере",
	"разгибание рук на блоке":                         "разгибание предплечья в кроссовере",
	"разгибание рук из-за головы в кроссовере":        "разгибание пред. из-за голов. в крос.",
	"разгибание предплечья из-за головы в кроссовере": "разгибание пред. из-за голов. в крос.",
	"разгибание предплечья из-за головы в крос.":      "разгибание пред. из-за голов. в крос.",
	"жим узким хватом":                                "жим лежа штанги узким хватом",
	"жим лежа узким хватом":                           "жим лежа штанги узким хватом",
	"поочередное разгибание предплечья в крос.":       "поочередное разгибание пред. в крос.",
	"тяга т-образного грифа тренажер":                 "тяга т-образного грифа в тренажере",
}

var allowedActivityPrefixes = []string{
	"отдых",
	"ходьба",
	"бег",
	"эллипс",
	"стэппер",
	"мобилити",
}

type trainingPlanDay struct {
	Day   int      `json:"day"`
	Name  string   `json:"name"`
	Focus string   `json:"focus"`
	Type  string   `json:"type"`
	Items []string `json:"items"`
}

type trainingPlanPayload struct {
	Week    []trainingPlanDay `json:"week_plan"`
	Comment string            `json:"comment,omitempty"`
}

func normalizeExerciseName(value string) string {
	out := strings.ToLower(strings.TrimSpace(value))
	if out == "" {
		return ""
	}
	out = strings.ReplaceAll(out, "ё", "е")
	for strings.Contains(out, "  ") {
		out = strings.ReplaceAll(out, "  ", " ")
	}
	return out
}

func extractExerciseName(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}
	for _, sep := range []string{" — ", " - ", " – "} {
		if idx := strings.Index(raw, sep); idx >= 0 {
			raw = raw[:idx]
			break
		}
	}
	normalized := normalizeExerciseName(raw)
	if alias, ok := exerciseAliases[normalized]; ok {
		return alias
	}
	return normalized
}

func isAllowedActivity(value string) bool {
	raw := normalizeExerciseName(value)
	if raw == "" {
		return false
	}
	for _, prefix := range allowedActivityPrefixes {
		if strings.HasPrefix(raw, prefix) {
			return true
		}
	}
	return false
}

func validateTrainingPlanPayload(planText string) []string {
	raw := strings.TrimSpace(planText)
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return nil
	}
	var payload trainingPlanPayload
	if err := json.Unmarshal([]byte(service.SanitizeJSON(raw)), &payload); err != nil {
		return nil
	}
	if len(payload.Week) == 0 {
		return nil
	}
	issues := make([]string, 0)
	for i, day := range payload.Week {
		name := strings.TrimSpace(day.Name)
		if name == "" || name == "—" {
			issues = append(issues, fmt.Sprintf("day_%d_no_name", i+1))
		}
		kind := strings.ToLower(strings.TrimSpace(day.Type))
		if kind == "" {
			issues = append(issues, fmt.Sprintf("day_%d_no_type", i+1))
		}
		for _, item := range day.Items {
			item = strings.TrimSpace(item)
			if item == "" {
				issues = append(issues, fmt.Sprintf("day_%d_empty_item", i+1))
				continue
			}
			if kind == "rest" {
				continue
			}
		}
	}
	return issues
}

func hasWeekPlanPayload(planText string) bool {
	raw := strings.TrimSpace(planText)
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return false
	}
	var payload trainingPlanPayload
	if err := json.Unmarshal([]byte(service.SanitizeJSON(raw)), &payload); err != nil {
		return strings.Contains(raw, "\"week_plan\"")
	}
	return len(payload.Week) > 0
}

func normalizeTrainingPlanTypes(planText string) string {
	raw := strings.TrimSpace(planText)
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return planText
	}
	var payload trainingPlanPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return planText
	}
	if len(payload.Week) == 0 {
		return planText
	}
	changed := false
	for i := range payload.Week {
		day := &payload.Week[i]
		if len(day.Items) == 0 {
			continue
		}
		allActivities := true
		for _, item := range day.Items {
			if !isAllowedActivity(item) {
				allActivities = false
				break
			}
		}
		if allActivities && len(day.Items) <= 2 {
			if strings.TrimSpace(strings.ToLower(day.Type)) != "rest" {
				day.Type = "rest"
				changed = true
			}
		}
	}
	if !changed {
		return planText
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return planText
	}
	return string(data)
}

func validateTrainingPlan(tp service.TrainingPlan) []string {
	issues := make([]string, 0)
	if len(tp.Days) < 7 {
		issues = append(issues, "days<7")
	}
	for i, day := range tp.Days {
		lines := splitPlanLines(day)
		if len(lines) > 0 {
			title := strings.TrimSpace(lines[0])
			if title == "" || strings.HasPrefix(title, "—") {
				issues = append(issues, fmt.Sprintf("day_%d_no_title", i+1))
			}
		}
		isRestDay := func() bool {
			if len(tp.Types) > i {
				kind := strings.ToLower(strings.TrimSpace(tp.Types[i]))
				if kind == "rest" {
					return true
				}
				if kind == "train" {
					return false
				}
			}
			lowered := strings.ToLower(day)
			if strings.Contains(lowered, "отдых") || strings.Contains(lowered, "rest") {
				return true
			}
			if strings.Contains(lowered, "мобилит") || strings.Contains(lowered, "ходьба") {
				return true
			}
			return false
		}()

		if len(tp.ExerciseCounts) > i && tp.ExerciseCounts[i] > 0 {
			if tp.ExerciseCounts[i] < 5 {
				if isRestDay && tp.ExerciseCounts[i] <= 2 {
					continue
				}
				issues = append(issues, fmt.Sprintf("day_%d_exercises<5", i+1))
			}
			continue
		}
		if len(lines) < 2 {
			issues = append(issues, fmt.Sprintf("day_%d_no_body", i+1))
			continue
		}
		count := 0
		for _, line := range lines[1:] {
			if exerciseLineRe.MatchString(line) {
				count++
			}
		}
		if count < 5 {
			if isRestDay && count <= 2 {
				continue
			}
			issues = append(issues, fmt.Sprintf("day_%d_exercises<5", i+1))
		}
	}
	return issues
}

func validateTrainingPlanWithProfile(tp service.TrainingPlan, desiredDays int, injuries, wishes string) []string {
	issues := make([]string, 0)
	if desiredDays >= 1 && desiredDays <= 7 {
		trainDays := countTrainDays(tp)
		if trainDays != desiredDays {
			issues = append(issues, fmt.Sprintf("train_days_%d_expected_%d", trainDays, desiredDays))
		}
		if desiredDays <= 5 && hasTrainStreak(tp, 3) {
			issues = append(issues, "train_streak>2")
		}
	}

	notes := strings.ToLower(strings.TrimSpace(injuries + " " + wishes))
	if needsBackCare(notes) {
		planText := strings.ToLower(strings.Join(tp.Days, "\n"))
		for _, term := range []string{
			"приседания со штангой",
			"приседания со штангой в смите",
			"становая",
			"мертвая тяга",
			"румынская тяга",
			"тяга штанги в наклоне",
		} {
			if strings.Contains(planText, term) {
				issues = append(issues, "back_loads")
				break
			}
		}
	}

	return issues
}

func hasTrainStreak(tp service.TrainingPlan, maxAllowed int) bool {
	streak := 0
	for i, day := range tp.Days {
		isTrain := false
		if len(tp.Types) > i {
			kind := strings.ToLower(strings.TrimSpace(tp.Types[i]))
			if kind == "train" {
				isTrain = true
			} else if kind == "rest" {
				isTrain = false
			}
		} else if len(tp.ExerciseCounts) > i && tp.ExerciseCounts[i] > 2 {
			isTrain = true
		} else {
			low := strings.ToLower(day)
			if strings.Contains(low, "отдых") || strings.Contains(low, "rest") || strings.Contains(low, "мобилит") || strings.Contains(low, "ходьба") {
				isTrain = false
			} else if strings.TrimSpace(day) != "" {
				isTrain = true
			}
		}
		if isTrain {
			streak++
			if streak >= maxAllowed {
				return true
			}
		} else {
			streak = 0
		}
	}
	return false
}

func countTrainDays(tp service.TrainingPlan) int {
	count := 0
	for i, day := range tp.Days {
		kind := ""
		if len(tp.Types) > i {
			kind = strings.ToLower(strings.TrimSpace(tp.Types[i]))
		}
		if kind == "train" {
			count++
			continue
		}
		if kind == "rest" {
			continue
		}
		if len(tp.ExerciseCounts) > i && tp.ExerciseCounts[i] > 2 {
			count++
			continue
		}
		low := strings.ToLower(day)
		if strings.Contains(low, "отдых") || strings.Contains(low, "rest") || strings.Contains(low, "мобилит") || strings.Contains(low, "ходьба") {
			continue
		}
		if strings.TrimSpace(day) != "" {
			count++
		}
	}
	return count
}

func needsBackCare(text string) bool {
	if text == "" {
		return false
	}
	for _, term := range []string{"поясн", "спин", "грыж", "межпозвон", "sciat", "back"} {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func splitPlanLines(text string) []string {
	raw := strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		clean := strings.TrimSpace(line)
		clean = strings.Trim(clean, "*_")
		if clean == "" {
			continue
		}
		out = append(out, clean)
	}
	return out
}

func mealsToDTO(items []db.Meal) []mealDTO {
	out := make([]mealDTO, 0, len(items))
	for _, m := range items {
		out = append(out, mealToDTO(m))
	}
	return out
}
