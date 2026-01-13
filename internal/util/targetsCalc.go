package util

import (
	"barzhafit/internal/domain"
	"math"
	"strconv"
	"strings"
)

func ActivityMultiplier(a string) float64 {
	a = strings.TrimSpace(strings.ToLower(a))
	if strings.HasPrefix(a, "ai:") {
		if v, err := strconv.ParseFloat(strings.TrimPrefix(a, "ai:"), 64); err == nil && v > 0 {
			return v
		}
	}
	if v, err := strconv.ParseFloat(a, 64); err == nil && v > 0 {
		return v
	}
	switch a {
	case "low":
		return 1.2
	case "high":
		return 1.6
	default:
		return 1.4 // mid
	}
}

// CalcTargets: Katch–McArdle (если есть %жира), иначе грубо по Mifflin (fallback)
func CalcTargets(p domain.Profile) domain.Targets {
	mult := ActivityMultiplier(p.Activity)

	var bmr float64
	var lbm float64

	if p.WeightKG > 0 && p.BodyFatPct > 0 && p.BodyFatPct < 60 {
		bf := p.BodyFatPct / 100.0
		lbm = p.WeightKG * (1.0 - bf)
		bmr = 370.0 + 21.6*lbm
	} else {
		// fallback: Mifflin-St Jeor (не идеал, но лучше чем 0)
		// BMR = 10W + 6.25H - 5A + S
		// S: +5 male, -161 female
		s := 0.0
		if p.Sex == "m" {
			s = 5
		} else if p.Sex == "f" {
			s = -161
		}
		bmr = 10.0*p.WeightKG + 6.25*float64(p.HeightCM) - 5.0*float64(p.Age) + s
		// lbm приблизим
		lbm = p.WeightKG * 0.8
	}

	tdee := bmr * mult
	kcal := int(math.Round(tdee))

	// макросы: белок/жир от веса
	baseWeight := p.WeightKG
	if baseWeight <= 0 {
		baseWeight = lbm
	}
	protein := int(math.Round(2.0 * baseWeight))
	fat := int(math.Round(1.0 * baseWeight))

	// carbs = остаток
	carbs := int(math.Round((float64(kcal) - float64(protein*4) - float64(fat*9)) / 4.0))
	if carbs < 0 {
		carbs = 0
	}

	return domain.Targets{
		ChatID:   p.ChatID,
		Kcal:     kcal,
		ProteinG: protein,
		FatG:     fat,
		CarbsG:   carbs,
		Steps:    10000,
		Source:   "calc",
	}
}
