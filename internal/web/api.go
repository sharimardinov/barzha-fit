package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"barzhafit/internal/domain"
	"barzhafit/internal/service"
	"barzhafit/internal/storage/db"
	"barzhafit/internal/util"
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
		Dislikes         string  `json:"dislikes"`
		CannotDo         string  `json:"cannot_do"`
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
		Dislikes:         trimLimit(payload.Dislikes, 200),
		CannotDo:         trimLimit(payload.CannotDo, 200),
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
	if s.ai == nil {
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

	payload := buildTrainingPrompt(p, tp)
	planText, raw, err := s.ai.GenerateTrainingPlan(ctx, payload)
	if err != nil {
		log.Printf("training generate failed: chat_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{
			OK:    false,
			Error: "training_generate_failed",
			Data:  raw,
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
	if s.ai == nil {
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
	mult, raw, err := s.ai.EstimateActivityMultiplierWithProfile(ctx, planText, p)
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
	Dislikes         string  `json:"dislikes"`
	CannotDo         string  `json:"cannot_do"`
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
		Dislikes:         p.Dislikes,
		CannotDo:         p.CannotDo,
		Wishes:           p.Wishes,
	}
}

type trainingPrompt struct {
	Sex           string  `json:"пол"`
	Age           int     `json:"возраст"`
	HeightCM      int     `json:"рост_см"`
	WeightKG      float64 `json:"вес_кг"`
	TrainingYears int     `json:"стаж_тренировок_лет"`
	BodyFatPct    float64 `json:"уровень_жира_проц,omitempty"`
	Strength      struct {
		BenchKG int     `json:"жим_лёжа_кг"`
		Pullups int     `json:"подтягивания_раз"`
		RunKM   float64 `json:"бег_км"`
	} `json:"силовые_показатели"`
	Injuries         string `json:"травмы"`
	Goal             string `json:"цель"`
	Pharma           string `json:"фармакология"`
	TrainingsPerWeek int    `json:"тренировок_в_неделю"`
	Preferences      string `json:"предпочтения"`
}

func buildTrainingPrompt(p domain.Profile, tp domain.TrainingProfile) trainingPrompt {
	sex := ""
	switch p.Sex {
	case "m":
		sex = "мужчина"
	case "f":
		sex = "женщина"
	}
	pharma := "нет"
	if tp.Pharma != nil && *tp.Pharma {
		pharma = "да"
	}

	out := trainingPrompt{
		Sex:              sex,
		Age:              p.Age,
		HeightCM:         p.HeightCM,
		WeightKG:         p.WeightKG,
		TrainingYears:    p.TrainingYears,
		BodyFatPct:       p.BodyFatPct,
		Injuries:         tp.Injuries,
		Goal:             tp.Goal,
		Pharma:           pharma,
		TrainingsPerWeek: tp.TrainingsPerWeek,
		Preferences:      "пожелания: " + tp.Wishes + "; не любит: " + tp.Dislikes + "; не может: " + tp.CannotDo,
	}
	out.Strength.BenchKG = tp.BenchKG
	out.Strength.Pullups = tp.Pullups
	out.Strength.RunKM = tp.RunKM
	return out
}

func missingTrainingFields(p domain.Profile, tp domain.TrainingProfile, hasProfile bool, hasTraining bool) []string {
	missing := make([]string, 0)
	if !hasProfile || p.Sex == "" {
		missing = append(missing, "пол")
	}
	if !hasProfile || p.Age <= 0 {
		missing = append(missing, "возраст")
	}
	if !hasProfile || p.HeightCM <= 0 {
		missing = append(missing, "рост_см")
	}
	if !hasProfile || p.WeightKG <= 0 {
		missing = append(missing, "вес_кг")
	}
	if !hasProfile || p.TrainingYears <= 0 {
		missing = append(missing, "стаж_тренировок_лет")
	}
	if !hasTraining || tp.BenchKG <= 0 {
		missing = append(missing, "жим_лёжа_кг")
	}
	if !hasTraining || tp.Pullups <= 0 {
		missing = append(missing, "подтягивания_раз")
	}
	if !hasTraining || tp.RunKM <= 0 {
		missing = append(missing, "бег_км")
	}
	if !hasTraining || strings.TrimSpace(tp.Injuries) == "" {
		missing = append(missing, "травмы")
	}
	if !hasTraining || strings.TrimSpace(tp.Goal) == "" {
		missing = append(missing, "цель")
	}
	if !hasTraining || tp.Pharma == nil {
		missing = append(missing, "фармакология")
	}
	if !hasTraining || tp.TrainingsPerWeek <= 0 {
		missing = append(missing, "тренировок_в_неделю")
	}
	if !hasTraining || strings.TrimSpace(tp.Dislikes) == "" {
		missing = append(missing, "что_не_любит")
	}
	if !hasTraining || strings.TrimSpace(tp.CannotDo) == "" {
		missing = append(missing, "что_не_может")
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

func mealsToDTO(items []db.Meal) []mealDTO {
	out := make([]mealDTO, 0, len(items))
	for _, m := range items {
		out = append(out, mealToDTO(m))
	}
	return out
}
