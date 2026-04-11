package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"barzhafit/backend/domain"
)

const (
	defaultWorkoutRestSec = 120
	betweenExtraRestSec   = 60
)

var (
	ErrWorkoutPlanNotFound      = errors.New("workout_plan_not_found")
	ErrWorkoutSessionNotFound   = errors.New("workout_session_not_found")
	ErrWorkoutSessionState      = errors.New("workout_session_state")
	ErrWorkoutSessionPaused     = errors.New("workout_session_paused")
)

type WorkoutPlanValidationError struct {
	Issues []string
}

func (e WorkoutPlanValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "workout_plan_invalid"
	}
	return fmt.Sprintf("workout_plan_invalid: %s", strings.Join(e.Issues, "; "))
}

type WorkoutPlanStorage interface {
	Get(ctx context.Context, chatID int64) (int64, []byte, bool, error)
	Upsert(ctx context.Context, chatID int64, payload []byte) (int64, error)
}

type WorkoutSessionStorage interface {
	GetActive(ctx context.Context, chatID int64) (domain.WorkoutSession, bool, error)
	GetByID(ctx context.Context, chatID, sessionID int64) (domain.WorkoutSession, bool, error)
	Create(ctx context.Context, s *domain.WorkoutSession) error
	Update(ctx context.Context, s *domain.WorkoutSession) error
}

type WorkoutSetStorage interface {
	Add(ctx context.Context, s *domain.WorkoutSet) error
	ListBySession(ctx context.Context, sessionID int64) ([]domain.WorkoutSet, error)
}

type WorkoutSessionExerciseReport struct {
	Name           string
	Type           domain.WorkoutExerciseType
	Sets           int
	TotalReps      int
	MaxWeight      float64
	TotalDurationSec int
}

type WorkoutSessionReport struct {
	Session          domain.WorkoutSession
	TotalDurationSec int
	Exercises        []WorkoutSessionExerciseReport
}

type WorkoutTimerService struct {
	plans    WorkoutPlanStorage
	sessions WorkoutSessionStorage
	sets     WorkoutSetStorage
	now      func() time.Time
}

func NewWorkoutTimerService(plans WorkoutPlanStorage, sessions WorkoutSessionStorage, sets WorkoutSetStorage) *WorkoutTimerService {
	return &WorkoutTimerService{
		plans:    plans,
		sessions: sessions,
		sets:     sets,
		now:      time.Now,
	}
}

func (s *WorkoutTimerService) GetPlan(ctx context.Context, chatID int64) (*domain.WorkoutPlan, int64, error) {
	id, payload, ok, err := s.plans.Get(ctx, chatID)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, ErrWorkoutPlanNotFound
	}
	plan, err := decodePlan(payload)
	if err != nil {
		return nil, 0, err
	}
	normalizePlan(plan)
	return plan, id, nil
}

func (s *WorkoutTimerService) SavePlan(ctx context.Context, chatID int64, plan *domain.WorkoutPlan) (int64, error) {
	if plan == nil {
		return 0, WorkoutPlanValidationError{Issues: []string{"plan_missing"}}
	}
	normalizePlan(plan)
	if err := validatePlan(plan); err != nil {
		return 0, err
	}
	payload, err := json.Marshal(plan)
	if err != nil {
		return 0, err
	}
	return s.plans.Upsert(ctx, chatID, payload)
}

func (s *WorkoutTimerService) StartSessionWithPlan(ctx context.Context, chatID int64, plan *domain.WorkoutPlan) (domain.WorkoutSession, bool, error) {
	session, ok, err := s.sessions.GetActive(ctx, chatID)
	if err != nil {
		return domain.WorkoutSession{}, false, err
	}
	if ok {
		parsed, err := decodePlan(session.PlanSnapshot)
		if err != nil {
			return domain.WorkoutSession{}, false, err
		}
		changed, err := s.advanceIfNeeded(ctx, &session, parsed)
		if err != nil {
			return domain.WorkoutSession{}, false, err
		}
		if changed {
			if err := s.sessions.Update(ctx, &session); err != nil {
				return domain.WorkoutSession{}, false, err
			}
		}
		return session, true, nil
	}
	if plan == nil {
		return domain.WorkoutSession{}, false, WorkoutPlanValidationError{Issues: []string{"plan_missing"}}
	}
	normalizePlan(plan)
	if err := validatePlan(plan); err != nil {
		return domain.WorkoutSession{}, false, err
	}
	payload, err := json.Marshal(plan)
	if err != nil {
		return domain.WorkoutSession{}, false, err
	}
	now := s.now()
	session = domain.WorkoutSession{
		ChatID:        chatID,
		PlanSnapshot:  payload,
		Status:        domain.WorkoutSessionStatusInProgress,
		Phase:         domain.WorkoutSessionPhaseWarmup,
		ExerciseIndex: 0,
		SetIndex:      0,
		StartedAt:     now,
	}
	if err := s.sessions.Create(ctx, &session); err != nil {
		return domain.WorkoutSession{}, false, err
	}
	return session, false, nil
}

func (s *WorkoutTimerService) StartSession(ctx context.Context, chatID int64) (domain.WorkoutSession, *domain.WorkoutPlan, bool, error) {
	session, ok, err := s.sessions.GetActive(ctx, chatID)
	if err != nil {
		return domain.WorkoutSession{}, nil, false, err
	}
	if ok {
		plan, err := decodePlan(session.PlanSnapshot)
		if err != nil {
			return domain.WorkoutSession{}, nil, false, err
		}
		changed, err := s.advanceIfNeeded(ctx, &session, plan)
		if err != nil {
			return domain.WorkoutSession{}, nil, false, err
		}
		if changed {
			if err := s.sessions.Update(ctx, &session); err != nil {
				return domain.WorkoutSession{}, nil, false, err
			}
		}
		return session, plan, true, nil
	}

	plan, planID, err := s.GetPlan(ctx, chatID)
	if err != nil {
		return domain.WorkoutSession{}, nil, false, err
	}
	payload, err := json.Marshal(plan)
	if err != nil {
		return domain.WorkoutSession{}, nil, false, err
	}
	now := s.now()
	session = domain.WorkoutSession{
		ChatID:        chatID,
		PlanID:        &planID,
		PlanSnapshot:  payload,
		Status:        domain.WorkoutSessionStatusInProgress,
		Phase:         domain.WorkoutSessionPhaseWarmup,
		ExerciseIndex: 0,
		SetIndex:      0,
		StartedAt:     now,
	}
	if err := s.sessions.Create(ctx, &session); err != nil {
		return domain.WorkoutSession{}, nil, false, err
	}
	return session, plan, false, nil
}

func (s *WorkoutTimerService) GetSession(ctx context.Context, chatID int64) (domain.WorkoutSession, *domain.WorkoutPlan, error) {
	session, ok, err := s.sessions.GetActive(ctx, chatID)
	if err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	if !ok {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionNotFound
	}
	plan, err := decodePlan(session.PlanSnapshot)
	if err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	changed, err := s.advanceIfNeeded(ctx, &session, plan)
	if err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	if changed {
		if err := s.sessions.Update(ctx, &session); err != nil {
			return domain.WorkoutSession{}, nil, err
		}
	}
	return session, plan, nil
}

func (s *WorkoutTimerService) StopSession(ctx context.Context, chatID int64) (domain.WorkoutSession, *domain.WorkoutPlan, error) {
	session, plan, err := s.GetSession(ctx, chatID)
	if err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	session.Status = domain.WorkoutSessionStatusCompleted
	session.Phase = domain.WorkoutSessionPhaseDone
	clearTimer(&session)
	if err := s.sessions.Update(ctx, &session); err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	return session, plan, nil
}

func (s *WorkoutTimerService) BuildSessionReport(ctx context.Context, chatID, sessionID int64) (WorkoutSessionReport, error) {
	session, ok, err := s.sessions.GetByID(ctx, chatID, sessionID)
	if err != nil {
		return WorkoutSessionReport{}, err
	}
	if !ok {
		return WorkoutSessionReport{}, ErrWorkoutSessionNotFound
	}

	sets, err := s.sets.ListBySession(ctx, sessionID)
	if err != nil {
		return WorkoutSessionReport{}, err
	}

	type aggregate struct {
		name             string
		kind             domain.WorkoutExerciseType
		sets             int
		totalReps        int
		maxWeight        float64
		totalDurationSec int
	}

	order := make([]int, 0)
	byExercise := make(map[int]*aggregate)
	for _, set := range sets {
		if set.IsWarmup {
			continue
		}
		item, exists := byExercise[set.ExerciseIndex]
		if !exists {
			item = &aggregate{
				name: set.ExerciseName,
				kind: set.ExerciseType,
			}
			byExercise[set.ExerciseIndex] = item
			order = append(order, set.ExerciseIndex)
		}
		item.sets++
		item.totalReps += set.ActualReps
		if set.ActualWeight > item.maxWeight {
			item.maxWeight = set.ActualWeight
		}
		if set.TargetWeight > item.maxWeight {
			item.maxWeight = set.TargetWeight
		}
		item.totalDurationSec += set.ActualDurationSec
		if set.ActualDurationSec <= 0 {
			item.totalDurationSec += set.TargetDurationSec
		}
	}

	exercises := make([]WorkoutSessionExerciseReport, 0, len(order))
	for _, exerciseIndex := range order {
		item := byExercise[exerciseIndex]
		exercises = append(exercises, WorkoutSessionExerciseReport{
			Name:             item.name,
			Type:             item.kind,
			Sets:             item.sets,
			TotalReps:        item.totalReps,
			MaxWeight:        item.maxWeight,
			TotalDurationSec: item.totalDurationSec,
		})
	}

	totalDurationSec := int(session.UpdatedAt.Sub(session.StartedAt).Seconds()) - session.PausedTotalSec
	if totalDurationSec < 0 {
		totalDurationSec = 0
	}

	return WorkoutSessionReport{
		Session:          session,
		TotalDurationSec: totalDurationSec,
		Exercises:        exercises,
	}, nil
}

func (s *WorkoutTimerService) FinishWarmup(ctx context.Context, chatID int64) (domain.WorkoutSession, *domain.WorkoutPlan, error) {
	session, plan, err := s.GetSession(ctx, chatID)
	if err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	if session.Status == domain.WorkoutSessionStatusPaused {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionPaused
	}
	if session.Phase != domain.WorkoutSessionPhaseWarmup {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionState
	}
	if len(plan.Exercises) == 0 {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionState
	}
	now := s.now()
	session.WarmupEndedAt = &now
	ex := plan.Exercises[session.ExerciseIndex]
	if ex.Type == domain.WorkoutExerciseCardio {
		startTimer(&session, domain.WorkoutSessionPhaseCardio, domain.WorkoutTimerKindCardio, ex.DurationSec, now)
		session.SetIndex = 1
	} else {
		session.Phase = domain.WorkoutSessionPhaseSet
		clearTimer(&session)
		session.SetIndex = 0
	}
	if err := s.sessions.Update(ctx, &session); err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	return session, plan, nil
}

func (s *WorkoutTimerService) FinishSet(ctx context.Context, chatID int64, exerciseIndex, setIndex int, actualWeight float64, actualReps int) (domain.WorkoutSession, *domain.WorkoutPlan, error) {
	session, plan, err := s.GetSession(ctx, chatID)
	if err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	if session.Status == domain.WorkoutSessionStatusPaused {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionPaused
	}
	if session.Phase != domain.WorkoutSessionPhaseSet {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionState
	}
	if session.ExerciseIndex != exerciseIndex || session.SetIndex != setIndex {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionState
	}
	if exerciseIndex < 0 || exerciseIndex >= len(plan.Exercises) {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionState
	}
	ex := plan.Exercises[exerciseIndex]
	if ex.Type != domain.WorkoutExerciseStrength {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionState
	}
	if ex.Sets <= 0 {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionState
	}

	now := s.now()
	isWarmup := setIndex == 0
	set := domain.WorkoutSet{
		SessionID:     session.ID,
		ExerciseIndex: exerciseIndex,
		SetIndex:      setIndex,
		IsWarmup:      isWarmup,
		ExerciseName:  ex.Name,
		ExerciseType:  ex.Type,
		TargetWeight:  ex.Weight,
		TargetReps:    ex.Reps,
		ActualWeight:  actualWeight,
		ActualReps:    actualReps,
		CompletedAt:   now,
	}
	if err := s.sets.Add(ctx, &set); err != nil {
		return domain.WorkoutSession{}, nil, err
	}

	restSec := restForExercise(plan, ex)
	if isWarmup {
		session.SetIndex = 1
		startTimer(&session, domain.WorkoutSessionPhaseRest, domain.WorkoutTimerKindRest, restSec, now)
	} else if setIndex < ex.Sets {
		session.SetIndex = setIndex + 1
		startTimer(&session, domain.WorkoutSessionPhaseRest, domain.WorkoutTimerKindRest, restSec, now)
	} else {
		nextIndex := exerciseIndex + 1
		if nextIndex >= len(plan.Exercises) {
			session.Status = domain.WorkoutSessionStatusCompleted
			session.Phase = domain.WorkoutSessionPhaseDone
			clearTimer(&session)
			if err := s.sessions.Update(ctx, &session); err != nil {
				return domain.WorkoutSession{}, nil, err
			}
			return session, plan, nil
		}
		next := plan.Exercises[nextIndex]
		session.ExerciseIndex = nextIndex
		if next.Type == domain.WorkoutExerciseCardio {
			session.SetIndex = 1
		} else {
			session.SetIndex = 0
		}
		restSec = restSec + betweenExtraRestSec
		startTimer(&session, domain.WorkoutSessionPhaseRest, domain.WorkoutTimerKindBetween, restSec, now)
	}

	if err := s.sessions.Update(ctx, &session); err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	return session, plan, nil
}

func (s *WorkoutTimerService) FinishRest(ctx context.Context, chatID int64) (domain.WorkoutSession, *domain.WorkoutPlan, error) {
	session, plan, err := s.GetSession(ctx, chatID)
	if err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	if session.Status == domain.WorkoutSessionStatusPaused {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionPaused
	}
	if session.Status != domain.WorkoutSessionStatusInProgress {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionState
	}
	if session.Phase != domain.WorkoutSessionPhaseRest {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionState
	}
	if session.ExerciseIndex < 0 || session.ExerciseIndex >= len(plan.Exercises) {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionState
	}

	now := s.now()
	ex := plan.Exercises[session.ExerciseIndex]
	if ex.Type == domain.WorkoutExerciseCardio {
		startTimer(&session, domain.WorkoutSessionPhaseCardio, domain.WorkoutTimerKindCardio, ex.DurationSec, now)
		session.SetIndex = 1
	} else {
		session.Phase = domain.WorkoutSessionPhaseSet
		clearTimer(&session)
	}

	if err := s.sessions.Update(ctx, &session); err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	return session, plan, nil
}

func (s *WorkoutTimerService) Pause(ctx context.Context, chatID int64) (domain.WorkoutSession, *domain.WorkoutPlan, error) {
	session, plan, err := s.GetSession(ctx, chatID)
	if err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	if session.Status == domain.WorkoutSessionStatusPaused {
		return session, plan, nil
	}
	now := s.now()
	if session.TimerStartedAt != nil && session.TimerDurationSec > 0 {
		elapsed := int(now.Sub(*session.TimerStartedAt).Seconds())
		remaining := session.TimerDurationSec - elapsed
		if remaining < 0 {
			remaining = 0
		}
		session.TimerDurationSec = remaining
		session.TimerStartedAt = nil
	}
	session.Status = domain.WorkoutSessionStatusPaused
	session.PausedAt = &now
	if err := s.sessions.Update(ctx, &session); err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	return session, plan, nil
}

func (s *WorkoutTimerService) Resume(ctx context.Context, chatID int64) (domain.WorkoutSession, *domain.WorkoutPlan, error) {
	session, ok, err := s.sessions.GetActive(ctx, chatID)
	if err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	if !ok {
		return domain.WorkoutSession{}, nil, ErrWorkoutSessionNotFound
	}
	plan, err := decodePlan(session.PlanSnapshot)
	if err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	if session.Status != domain.WorkoutSessionStatusPaused {
		return session, plan, nil
	}
	now := s.now()
	if session.PausedAt != nil {
		pausedFor := int(now.Sub(*session.PausedAt).Seconds())
		if pausedFor > 0 {
			session.PausedTotalSec += pausedFor
		}
	}
	session.PausedAt = nil
	session.Status = domain.WorkoutSessionStatusInProgress
	if session.TimerStartedAt == nil && session.TimerDurationSec > 0 && session.Phase == domain.WorkoutSessionPhaseRest {
		startTimer(&session, session.Phase, session.TimerKind, session.TimerDurationSec, now)
	}
	if session.TimerStartedAt == nil && session.TimerDurationSec > 0 && session.Phase == domain.WorkoutSessionPhaseCardio {
		startTimer(&session, session.Phase, domain.WorkoutTimerKindCardio, session.TimerDurationSec, now)
	}
	if err := s.sessions.Update(ctx, &session); err != nil {
		return domain.WorkoutSession{}, nil, err
	}
	return session, plan, nil
}

func (s *WorkoutTimerService) advanceIfNeeded(ctx context.Context, session *domain.WorkoutSession, plan *domain.WorkoutPlan) (bool, error) {
	if session.Status != domain.WorkoutSessionStatusInProgress {
		return false, nil
	}
	changed := false
	for i := 0; i < 5; i++ {
		if session.TimerStartedAt == nil || session.TimerDurationSec <= 0 {
			return changed, nil
		}
		now := s.now()
		if now.Before(session.TimerStartedAt.Add(time.Duration(session.TimerDurationSec) * time.Second)) {
			return changed, nil
		}

		switch session.Phase {
		case domain.WorkoutSessionPhaseRest:
			if session.ExerciseIndex < 0 || session.ExerciseIndex >= len(plan.Exercises) {
				return changed, ErrWorkoutSessionState
			}
			ex := plan.Exercises[session.ExerciseIndex]
			if ex.Type == domain.WorkoutExerciseCardio {
				startTimer(session, domain.WorkoutSessionPhaseCardio, domain.WorkoutTimerKindCardio, ex.DurationSec, now)
				session.SetIndex = 1
			} else {
				session.Phase = domain.WorkoutSessionPhaseSet
				clearTimer(session)
			}
			changed = true
			continue
		case domain.WorkoutSessionPhaseCardio:
			if session.ExerciseIndex < 0 || session.ExerciseIndex >= len(plan.Exercises) {
				return changed, ErrWorkoutSessionState
			}
			ex := plan.Exercises[session.ExerciseIndex]
			set := domain.WorkoutSet{
				SessionID:         session.ID,
				ExerciseIndex:     session.ExerciseIndex,
				SetIndex:          1,
				ExerciseName:      ex.Name,
				ExerciseType:      ex.Type,
				TargetDurationSec: ex.DurationSec,
				ActualDurationSec: ex.DurationSec,
				CompletedAt:       now,
			}
			if err := s.sets.Add(ctx, &set); err != nil {
				return changed, err
			}
			nextIndex := session.ExerciseIndex + 1
			if nextIndex >= len(plan.Exercises) {
				session.Status = domain.WorkoutSessionStatusCompleted
				session.Phase = domain.WorkoutSessionPhaseDone
				clearTimer(session)
				changed = true
				return changed, nil
			}
			session.ExerciseIndex = nextIndex
			next := plan.Exercises[nextIndex]
			if next.Type == domain.WorkoutExerciseCardio {
				session.SetIndex = 1
			} else {
				session.SetIndex = 0
			}
			restSec := restForExercise(plan, ex) + betweenExtraRestSec
			startTimer(session, domain.WorkoutSessionPhaseRest, domain.WorkoutTimerKindBetween, restSec, now)
			changed = true
			continue
		default:
			return changed, nil
		}
	}
	return changed, nil
}

func decodePlan(payload []byte) (*domain.WorkoutPlan, error) {
	var plan domain.WorkoutPlan
	if err := json.Unmarshal(payload, &plan); err != nil {
		return nil, err
	}
	normalizePlan(&plan)
	return &plan, nil
}

func normalizePlan(plan *domain.WorkoutPlan) {
	if plan.DefaultRestSec <= 0 {
		plan.DefaultRestSec = defaultWorkoutRestSec
	}
	for i := range plan.Exercises {
		ex := &plan.Exercises[i]
		ex.Name = strings.TrimSpace(ex.Name)
		if ex.Type == "" {
			ex.Type = domain.WorkoutExerciseStrength
		}
		if ex.RestSec <= 0 {
			ex.RestSec = plan.DefaultRestSec
		}
	}
}

func validatePlan(plan *domain.WorkoutPlan) error {
	issues := make([]string, 0)
	if len(plan.Exercises) == 0 {
		issues = append(issues, "no_exercises")
	}
	for i, ex := range plan.Exercises {
		if ex.Name == "" {
			issues = append(issues, fmt.Sprintf("ex[%d].name_empty", i))
		}
		switch ex.Type {
		case domain.WorkoutExerciseCardio:
			if ex.DurationSec <= 0 {
				issues = append(issues, fmt.Sprintf("ex[%d].duration_missing", i))
			}
		case domain.WorkoutExerciseStrength:
			if ex.Sets <= 0 {
				issues = append(issues, fmt.Sprintf("ex[%d].sets_missing", i))
			}
			if ex.Reps <= 0 {
				issues = append(issues, fmt.Sprintf("ex[%d].reps_missing", i))
			}
			if ex.Weight < 0 {
				issues = append(issues, fmt.Sprintf("ex[%d].weight_negative", i))
			}
		default:
			issues = append(issues, fmt.Sprintf("ex[%d].type_invalid", i))
		}
		if ex.RestSec < 0 {
			issues = append(issues, fmt.Sprintf("ex[%d].rest_negative", i))
		}
	}
	if len(issues) > 0 {
		return WorkoutPlanValidationError{Issues: issues}
	}
	return nil
}

func restForExercise(plan *domain.WorkoutPlan, ex domain.WorkoutExercise) int {
	if ex.RestSec > 0 {
		return ex.RestSec
	}
	if plan.DefaultRestSec > 0 {
		return plan.DefaultRestSec
	}
	return defaultWorkoutRestSec
}

func startTimer(session *domain.WorkoutSession, phase domain.WorkoutSessionPhase, kind domain.WorkoutTimerKind, durationSec int, now time.Time) {
	session.Phase = phase
	session.TimerKind = kind
	session.TimerDurationSec = durationSec
	session.TimerStartedAt = &now
}

func clearTimer(session *domain.WorkoutSession) {
	session.TimerKind = ""
	session.TimerDurationSec = 0
	session.TimerStartedAt = nil
}
