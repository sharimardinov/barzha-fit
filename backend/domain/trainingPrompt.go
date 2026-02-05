package domain

import "strings"

type TrainingPrompt struct {
	Sex           string  `json:"sex"`
	Age           int     `json:"age"`
	HeightCM      int     `json:"heightCm"`
	WeightKG      float64 `json:"weightKg"`
	TrainingYears int     `json:"trainingYears"`
	BodyFatPct    float64 `json:"bodyFatPct,omitempty"`
	Strength      struct {
		BenchKG int     `json:"benchKg"`
		Pullups int     `json:"pullups"`
		RunKM   float64 `json:"runKm"`
	} `json:"strength"`
	Injuries         string `json:"injuries"`
	Goal             string `json:"goal"`
	Pharma           string `json:"pharma"`
	TrainingsPerWeek int    `json:"trainingsPerWeek"`
	Preferences      string `json:"preferences"`
	Normalized       struct {
		TrainingDaysPerWeek int      `json:"trainingDaysPerWeek"`
		Goal                string   `json:"goal"`
		Equipment           string   `json:"equipment"`
		Injuries            []string `json:"injuries"`
		Experience          string   `json:"experience"`
		MainLifts           struct {
			BenchKG     int     `json:"benchKg"`
			PullupsReps int     `json:"pullupsReps"`
			RunKM       float64 `json:"runKm"`
		} `json:"mainLifts"`
		Preferences struct {
			Notes string `json:"notes"`
		} `json:"preferences"`
	} `json:"normalized"`
}

func BuildTrainingPrompt(p Profile, tp TrainingProfile) TrainingPrompt {
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
	injuries := strings.TrimSpace(tp.Injuries)
	if injuries == "" {
		injuries = "травм нет"
	}
	wishes := strings.TrimSpace(tp.Wishes)
	if wishes == "" {
		wishes = "пожеланий нет"
	}
	experience := "beginner"
	switch {
	case p.TrainingYears >= 4:
		experience = "advanced"
	case p.TrainingYears >= 1:
		experience = "intermediate"
	}

	injuryList := make([]string, 0)
	if injuries != "травм нет" {
		for _, part := range strings.FieldsFunc(injuries, func(r rune) bool {
			return r == ',' || r == ';'
		}) {
			item := strings.TrimSpace(part)
			if item != "" {
				injuryList = append(injuryList, item)
			}
		}
	}

	out := TrainingPrompt{
		Sex:              sex,
		Age:              p.Age,
		HeightCM:         p.HeightCM,
		WeightKG:         p.WeightKG,
		TrainingYears:    p.TrainingYears,
		BodyFatPct:       p.BodyFatPct,
		Injuries:         injuries,
		Goal:             tp.Goal,
		Pharma:           pharma,
		TrainingsPerWeek: tp.TrainingsPerWeek,
		Preferences:      "пожелания: " + wishes,
	}
	out.Strength.BenchKG = tp.BenchKG
	out.Strength.Pullups = tp.Pullups
	out.Strength.RunKM = tp.RunKM
	out.Normalized.TrainingDaysPerWeek = tp.TrainingsPerWeek
	out.Normalized.Goal = tp.Goal
	out.Normalized.Equipment = "unknown"
	out.Normalized.Injuries = injuryList
	out.Normalized.Experience = experience
	out.Normalized.MainLifts.BenchKG = tp.BenchKG
	out.Normalized.MainLifts.PullupsReps = tp.Pullups
	out.Normalized.MainLifts.RunKM = tp.RunKM
	out.Normalized.Preferences.Notes = wishes
	return out
}
