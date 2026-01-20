package domain

import "time"

type StrengthTotals struct {
	Sets      int     `json:"sets"`
	Reps      int     `json:"reps"`
	Tonnage   float64 `json:"tonnage"`
	AvgWeight float64 `json:"avgWeight"`
	AvgReps   float64 `json:"avgReps"`
	MaxWeight float64 `json:"maxWeight"`
}

type StrengthExerciseStats struct {
	Name      string  `json:"name"`
	Sets      int     `json:"sets"`
	Reps      int     `json:"reps"`
	Tonnage   float64 `json:"tonnage"`
	MaxWeight float64 `json:"maxWeight"`
}

type StrengthExerciseEntry struct {
	Weight      float64   `json:"weight"`
	Reps        int       `json:"reps"`
	CompletedAt time.Time `json:"completedAt"`
}

type StrengthExerciseRecent struct {
	Name    string                  `json:"name"`
	Entries []StrengthExerciseEntry `json:"entries"`
}

type StrengthStats struct {
	Totals       StrengthTotals           `json:"totals"`
	TopByTonnage []StrengthExerciseStats  `json:"topByTonnage"`
	TopByReps    []StrengthExerciseStats  `json:"topByReps"`
	Recent       []StrengthExerciseRecent `json:"recent"`
}
