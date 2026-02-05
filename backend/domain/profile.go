package domain

import "strings"

// Sex represents biological sex
type Sex string

const (
	SexMale    Sex = "m"
	SexFemale  Sex = "f"
	SexUnknown Sex = ""
)

// ActivityLevel represents physical activity level
type ActivityLevel string

const (
	ActivityLow  ActivityLevel = "low"
	ActivityMid  ActivityLevel = "mid"
	ActivityHigh ActivityLevel = "high"
)

// ProfileGoal represents the user's fitness goal
type ProfileGoal string

const (
	ProfileGoalCut     ProfileGoal = "cut"
	ProfileGoalBalance ProfileGoal = "balance"
	ProfileGoalBulk    ProfileGoal = "bulk"
)

type Profile struct {
	ChatID        int64
	Sex           string // "m"|"f"|"" (unknown)
	Age           int
	HeightCM      int
	WeightKG      float64
	BodyFatPct    float64
	Activity      string // "low"|"mid"|"high"
	Goal          string // "cut"|"balance"|"bulk"
	TrainingYears int
}

// ValidationError represents a validation error with field name
type ValidationError struct {
	Field   string
	Message string
}

// Validate checks if the profile has valid data
func (p Profile) Validate() []ValidationError {
	var errors []ValidationError

	if p.Sex != "" && p.Sex != string(SexMale) && p.Sex != string(SexFemale) {
		errors = append(errors, ValidationError{Field: "sex", Message: "must be 'm', 'f', or empty"})
	}

	if p.Age != 0 && (p.Age < 14 || p.Age > 100) {
		errors = append(errors, ValidationError{Field: "age", Message: "must be between 14 and 100"})
	}

	if p.HeightCM != 0 && (p.HeightCM < 100 || p.HeightCM > 250) {
		errors = append(errors, ValidationError{Field: "height_cm", Message: "must be between 100 and 250"})
	}

	if p.WeightKG != 0 && (p.WeightKG < 30 || p.WeightKG > 300) {
		errors = append(errors, ValidationError{Field: "weight_kg", Message: "must be between 30 and 300"})
	}

	if p.BodyFatPct != 0 && (p.BodyFatPct < 3 || p.BodyFatPct > 60) {
		errors = append(errors, ValidationError{Field: "bodyfat_pct", Message: "must be between 3 and 60"})
	}

	if p.Activity != "" {
		act := strings.ToLower(p.Activity)
		if act != string(ActivityLow) && act != string(ActivityMid) && act != string(ActivityHigh) && !strings.HasPrefix(act, "ai:") {
			errors = append(errors, ValidationError{Field: "activity", Message: "must be 'low', 'mid', 'high', or 'ai:X.XX'"})
		}
	}

	if p.Goal != "" {
		goal := strings.ToLower(p.Goal)
		if goal != string(ProfileGoalCut) && goal != string(ProfileGoalBalance) && goal != string(ProfileGoalBulk) {
			errors = append(errors, ValidationError{Field: "goal", Message: "must be 'cut', 'balance', or 'bulk'"})
		}
	}

	if p.TrainingYears < 0 || p.TrainingYears > 50 {
		errors = append(errors, ValidationError{Field: "training_years", Message: "must be between 0 and 50"})
	}

	return errors
}

// IsValid returns true if the profile passes all validations
func (p Profile) IsValid() bool {
	return len(p.Validate()) == 0
}
