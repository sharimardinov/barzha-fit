package domain

import "strings"

type TrainingPrompt struct {
	Sex           string  `json:"пол"`
	Age           int     `json:"возраст"`
	HeightCM      int     `json:"рост_см"`
	WeightKG      float64 `json:"вес_кг"`
	TrainingYears int     `json:"стаж_тренировок_лет"`
	BodyFatPct    float64 `json:"уровень_жира_проц,omitempty"`
	Strength      struct {
		BenchKG int     `json:"жим_лёжа_кг"`
		Pullups int     `json:"подтягивания_раз"`
		RunKM   float64 `json:"бег_км"`
	} `json:"силовые_показатели"`
	Injuries         string `json:"травмы"`
	Goal             string `json:"цель"`
	Pharma           string `json:"фармакология"`
	TrainingsPerWeek int    `json:"тренировок_в_неделю"`
	Preferences      string `json:"предпочтения"`
	Normalized       struct {
		TrainingDaysPerWeek int      `json:"training_days_per_week"`
		Goal                string   `json:"goal"`
		Equipment           string   `json:"equipment"`
		Injuries            []string `json:"injuries"`
		Experience          string   `json:"experience"`
		MainLifts           struct {
			BenchKG     int     `json:"bench_kg"`
			PullupsReps int     `json:"pullups_reps"`
			RunKM       float64 `json:"run_km"`
		} `json:"main_lifts"`
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
