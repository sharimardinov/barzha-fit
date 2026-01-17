package domain

import (
	"strings"
	"time"
)

type FitnessLevel string

const (
	FitnessBeginner     FitnessLevel = "beginner"
	FitnessIntermediate FitnessLevel = "intermediate"
	FitnessAdvanced     FitnessLevel = "advanced"
)

type FitnessGoal string

const (
	GoalHypertrophy FitnessGoal = "hypertrophy"
	GoalStrength    FitnessGoal = "strength"
	GoalFatLoss     FitnessGoal = "fat_loss"
)

type TrainingInput struct {
	ID          string
	UserID      string
	Level       FitnessLevel
	Goal        FitnessGoal
	DaysPerWeek int
	Injuries    []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NormalizeFitnessLevel(raw string) (FitnessLevel, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "core", string(FitnessBeginner):
		return FitnessBeginner, true
	case "flow", string(FitnessIntermediate):
		return FitnessIntermediate, true
	case "peak", string(FitnessAdvanced):
		return FitnessAdvanced, true
	default:
		return "", false
	}
}

func NormalizeFitnessGoal(raw string) (FitnessGoal, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "cut", string(GoalFatLoss):
		return GoalFatLoss, true
	case "bulk", string(GoalHypertrophy):
		return GoalHypertrophy, true
	case "balance", string(GoalStrength):
		return GoalStrength, true
	default:
		return "", false
	}
}
