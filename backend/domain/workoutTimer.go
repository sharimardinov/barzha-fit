package domain

import "time"

// WorkoutExerciseType represents the type of exercise
type WorkoutExerciseType string

const (
	WorkoutExerciseStrength WorkoutExerciseType = "strength"
	WorkoutExerciseCardio   WorkoutExerciseType = "cardio"
)

type WorkoutPlan struct {
	DefaultRestSec int               `json:"defaultRestSec"`
	Exercises      []WorkoutExercise `json:"exercises"`
}

type WorkoutExercise struct {
	Name        string              `json:"name"`
	Type        WorkoutExerciseType `json:"type"`
	Weight      float64             `json:"weight,omitempty"`
	Reps        int                 `json:"reps,omitempty"`
	Sets        int                 `json:"sets,omitempty"`
	RestSec     int                 `json:"restSec,omitempty"`
	DurationSec int                 `json:"durationSec,omitempty"`
}

// WorkoutSessionStatus represents the status of a workout session
type WorkoutSessionStatus string

const (
	WorkoutSessionStatusInProgress WorkoutSessionStatus = "in_progress"
	WorkoutSessionStatusPaused     WorkoutSessionStatus = "paused"
	WorkoutSessionStatusCompleted  WorkoutSessionStatus = "completed"
)

// WorkoutSessionPhase represents the current phase of a workout session
type WorkoutSessionPhase string

const (
	WorkoutSessionPhaseWarmup WorkoutSessionPhase = "warmup"
	WorkoutSessionPhaseSet    WorkoutSessionPhase = "set"
	WorkoutSessionPhaseRest   WorkoutSessionPhase = "rest"
	WorkoutSessionPhaseCardio WorkoutSessionPhase = "cardio"
	WorkoutSessionPhaseDone   WorkoutSessionPhase = "done"
)

// WorkoutTimerKind represents the type of timer being used
type WorkoutTimerKind string

const (
	WorkoutTimerKindRest    WorkoutTimerKind = "rest"
	WorkoutTimerKindBetween WorkoutTimerKind = "between"
	WorkoutTimerKindCardio  WorkoutTimerKind = "cardio"
)

type WorkoutSession struct {
	ID               int64
	ChatID           int64
	PlanID           *int64
	PlanSnapshot     []byte
	Status           WorkoutSessionStatus
	Phase            WorkoutSessionPhase
	ExerciseIndex    int
	SetIndex         int
	TimerKind        WorkoutTimerKind
	TimerStartedAt   *time.Time
	TimerDurationSec int
	WarmupEndedAt    *time.Time
	PausedAt         *time.Time
	PausedTotalSec   int
	StartedAt        time.Time
	UpdatedAt        time.Time
}

type WorkoutSet struct {
	ID               int64
	SessionID        int64
	ExerciseIndex    int
	SetIndex         int
	IsWarmup         bool
	ExerciseName     string
	ExerciseType     WorkoutExerciseType
	TargetWeight     float64
	TargetReps       int
	TargetDurationSec int
	ActualWeight     float64
	ActualReps       int
	ActualDurationSec int
	CompletedAt      time.Time
}
