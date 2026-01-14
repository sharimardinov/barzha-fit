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
}
