package domain

type TrainingProfile struct {
	ChatID           int64
	BenchKG          int
	Pullups          int
	RunKM            float64
	Injuries         string
	Goal             string
	Pharma           *bool
	TrainingsPerWeek int
	Dislikes         string
	CannotDo         string
	Wishes           string
}

// Validate checks if the training profile has valid data
func (tp TrainingProfile) Validate() []ValidationError {
	var errors []ValidationError

	if tp.BenchKG < 0 || tp.BenchKG > 500 {
		errors = append(errors, ValidationError{Field: "bench_kg", Message: "must be between 0 and 500"})
	}

	if tp.Pullups < 0 || tp.Pullups > 100 {
		errors = append(errors, ValidationError{Field: "pullups", Message: "must be between 0 and 100"})
	}

	if tp.RunKM < 0 || tp.RunKM > 100 {
		errors = append(errors, ValidationError{Field: "run_km", Message: "must be between 0 and 100"})
	}

	if tp.TrainingsPerWeek < 0 || tp.TrainingsPerWeek > 7 {
		errors = append(errors, ValidationError{Field: "trainings_per_week", Message: "must be between 0 and 7"})
	}

	return errors
}

// IsValid returns true if the training profile passes all validations
func (tp TrainingProfile) IsValid() bool {
	return len(tp.Validate()) == 0
}
