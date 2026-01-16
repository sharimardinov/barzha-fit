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
	mux.HandleFunc("/api/training/generate", s.withAuth(s.handleTrainingGenerate))
	mux.HandleFunc("/api/weight/set", s.withAuth(s.handleWeightSet))
	mux.HandleFunc("/api/stats/week", s.withAuth(s.handleStatsWeek))
	mux.HandleFunc("/api/stats/month", s.withAuth(s.handleStatsMonth))
	mux.HandleFunc("/api/streak", s.withAuth(s.handleStreak))
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
			issues := validateTrainingPlan(tp)
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
	if payload.TrainingYears > 0 {
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

func (s *Server) handleTrainingGenerate(w http.ResponseWriter, r *http.Request, auth authContext) {
	if s.trainingAI == nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "training_ai_unavailable"})
		return
	}
	if s.training == nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "training_profile_unavailable"})
		return
	}
	ctx := context.Background()
	p, ok, err := s.profile.Get(ctx, auth.User.ID)
	if err != nil {
		log.Printf("training profile read user failed: chat_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "profile_read_failed"})
		return
	}
	tp, okTP, err := s.training.Get(ctx, auth.User.ID)
	if err != nil {
		log.Printf("training profile read failed: chat_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "training_profile_read_failed"})
		return
	}

	missing := missingTrainingFields(p, tp, ok, okTP)
	if len(missing) > 0 {
		writeJSON(w, http.StatusBadRequest, apiResponse{
			OK:    false,
			Error: "missing_fields",
			Data:  map[string]any{"fields": missing},
		})
		return
	}

	payload := domain.BuildTrainingPrompt(p, tp)
	var planText string
	var raw any
	var issues []string
	for attempt := 0; attempt < 3; attempt++ {
		planText, raw, err = s.trainingAI.GenerateTrainingPlan(ctx, payload)
		if err != nil {
			log.Printf("training generate failed: chat_id=%d err=%v", auth.User.ID, err)
			writeJSON(w, http.StatusInternalServerError, apiResponse{
				OK:    false,
				Error: "training_generate_failed",
				Data:  raw,
			})
			return
		}
		normalized, ok := service.NormalizeTrainingPlan(planText)
		if !ok {
			issues = []string{"invalid_json"}
		} else {
			planText = normalizeTrainingPlanTypes(normalized)
			if tp, ok := service.ParseTrainingPlan(planText); ok {
				issues = validateTrainingPlan(tp)
				if payloadIssues := validateTrainingPlanPayload(planText); len(payloadIssues) > 0 {
					issues = append(issues, payloadIssues...)
				}
			} else {
				issues = []string{"invalid_json"}
			}
		}
		if len(issues) == 0 {
			break
		}
		if attempt < 2 {
			log.Printf("training plan retry: chat_id=%d attempt=%d issues=%v", auth.User.ID, attempt+1, issues)
		}
	}
	if len(issues) > 0 {
		snippet := planText
		if len(snippet) > 600 {
			snippet = snippet[:600] + "..."
		}
		log.Printf("training plan invalid: chat_id=%d issues=%v plan=%s", auth.User.ID, issues, snippet)
		writeJSON(w, http.StatusUnprocessableEntity, apiResponse{
			OK:    false,
			Error: "training_plan_invalid",
			Data:  map[string]any{"issues": issues},
		})
		return
	}

	if err := s.plan.Save(ctx, auth.User.ID, planText); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "plan_save_failed"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"plan": planText}})
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
	ctx := context.Background()
	loc := util.MustLocation(s.tz)
	now := util.NowIn(loc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
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
		"offset": offset,
		"days":   days,
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
	if !hasProfile || p.TrainingYears <= 0 {
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

type trainingPlanPayload struct {
	Week []struct {
		Day   int      `json:"day"`
		Name  string   `json:"name"`
		Focus string   `json:"focus"`
		Type  string   `json:"type"`
		Items []string `json:"items"`
	} `json:"week_plan"`
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
