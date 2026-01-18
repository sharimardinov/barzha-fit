package util

import (
	"barzhafit/backend/domain"
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

func CalcTargets(p domain.Profile) domain.Targets {
	mult := ActivityMultiplier(p.Activity)

	var bmr float64
	var lbm float64

	if p.WeightKG > 0 && p.BodyFatPct > 0 && p.BodyFatPct < 60 {
		bf := p.BodyFatPct / 100.0
		lbm = p.WeightKG * (1.0 - bf)
		bmr = 370.0 + 21.6*lbm
	} else {
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
	switch p.Goal {
	case "cut":
		kcal = int(math.Round(float64(kcal) * 0.8))
	case "bulk":
		kcal = int(math.Round(float64(kcal) * 1.1))
	default:
	}

	// макросы: белок от сухой массы (если есть), жир от общей массы
	proteinWeight := lbm
	if proteinWeight <= 0 {
		proteinWeight = p.WeightKG
	}
	proteinRate := 1.8
	switch p.Goal {
	case "cut":
		proteinRate = 2.2
	case "bulk":
		proteinRate = 2.0
	default:
		proteinRate = 1.8
	}
	protein := int(math.Round(proteinRate * proteinWeight))

	fatWeight := p.WeightKG
	if fatWeight <= 0 {
		fatWeight = lbm
	}
	fatRate := 0.9
	switch p.Goal {
	case "cut":
		fatRate = 0.9
	case "bulk":
		fatRate = 1.05
	default:
		fatRate = 0.9
	}
	fat := int(math.Round(fatRate * fatWeight))
	if fat < int(math.Round(0.6*fatWeight)) {
		fat = int(math.Round(0.6 * fatWeight))
	}
	if fat < 40 {
		fat = 40
	}

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
