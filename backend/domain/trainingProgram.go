package domain

import (
	"encoding/json"
	"time"
)

type ProgramTemplate struct {
	ID        string
	Name      string
	Days      []string
	Structure TemplateStructure
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TemplateStructure struct {
	Days []TemplateDay `json:"days"`
}

type TemplateDay struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	MuscleGroups []string `json:"muscle_groups"`
}

type Exercise struct {
	ID                string
	Name              string
	MuscleGroup       string
	Type              []string
	Level             []FitnessLevel
	Priority          string
	Contraindications []string
	SubstituteFor     []string
	PrehabTarget      string
}

type Periodization struct {
	Week       int    `json:"week"`
	Intensity  string `json:"intensity"`
	Percent1RM string `json:"percent_1rm"`
	Reps       string `json:"reps"`
	Rest       string `json:"rest"`
}

type UserProgram struct {
	ID            string
	UserID        string
	TemplateID    string
	StartDate     time.Time
	CurrentWeek   int
	DaysGenerated json.RawMessage
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type GeneratedProgram struct {
	Template      string         `json:"template"`
	Week          int            `json:"week"`
	Periodization Periodization  `json:"periodization"`
	Days          []GeneratedDay `json:"days"`
}

type GeneratedDay struct {
	Day       int                 `json:"day"`
	Name      string              `json:"name"`
	Focus     string              `json:"focus"`
	Type      string              `json:"type"`
	Exercises []GeneratedExercise `json:"exercises"`
}

type GeneratedExercise struct {
	ExerciseID  string   `json:"exercise_id"`
	Name        string   `json:"name"`
	MuscleGroup string   `json:"muscle_group"`
	Priority    string   `json:"priority"`
	Sets        int      `json:"sets"`
	Reps        string   `json:"reps"`
	RPE         string   `json:"rpe"`
	Rest        string   `json:"rest"`
	Percent1RM  string   `json:"percent_1rm,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}
