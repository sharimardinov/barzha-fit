package domain

import "time"

const (
	WorkoutExerciseStrength = "strength"
	WorkoutExerciseCardio   = "cardio"
)

type WorkoutPlan struct {
	DefaultRestSec int               `json:"defaultRestSec"`
	Exercises      []WorkoutExercise `json:"exercises"`
}

type WorkoutExercise struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Weight      float64 `json:"weight,omitempty"`
	Reps        int     `json:"reps,omitempty"`
	Sets        int     `json:"sets,omitempty"`
	RestSec     int     `json:"restSec,omitempty"`
	DurationSec int     `json:"durationSec,omitempty"`
}

const (
	WorkoutSessionStatusInProgress = "in_progress"
	WorkoutSessionStatusPaused     = "paused"
	WorkoutSessionStatusCompleted  = "completed"
)

const (
	WorkoutSessionPhaseWarmup = "warmup"
	WorkoutSessionPhaseSet    = "set"
	WorkoutSessionPhaseRest   = "rest"
	WorkoutSessionPhaseCardio = "cardio"
	WorkoutSessionPhaseDone   = "done"
)

const (
	WorkoutTimerKindRest    = "rest"
	WorkoutTimerKindBetween = "between"
	WorkoutTimerKindCardio  = "cardio"
)

type WorkoutSession struct {
	ID               int64
	ChatID           int64
	PlanID           *int64
	PlanSnapshot     []byte
	Status           string
	Phase            string
	ExerciseIndex    int
	SetIndex         int
	TimerKind        string
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
	ExerciseType     string
	TargetWeight     float64
	TargetReps       int
	TargetDurationSec int
	ActualWeight     float64
	ActualReps       int
	ActualDurationSec int
	CompletedAt      time.Time
}
