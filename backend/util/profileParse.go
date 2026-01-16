package util

import (
	"barzhafit/backend/domain"
	"regexp"
	"strconv"
	"strings"
)

var (
	reHeight = regexp.MustCompile(`(?i)(рост|height)\s*[:=]?\s*(\d{2,3})`)
	reWeight = regexp.MustCompile(`(?i)(вес|weight)\s*[:=]?\s*(\d{2,3}([.,]\d)?)`)
	reBF     = regexp.MustCompile(`(?i)(жир|bf|bodyfat)\s*[:=]?\s*(\d{1,2}([.,]\d)?)`)
	reAge    = regexp.MustCompile(`(?i)(возраст|age)\s*[:=]?\s*(\d{1,2})`)
	reSex    = regexp.MustCompile(`(?i)(пол|sex)\s*[:=]?\s*(m|f|м|ж|male|female)`)
	reAct    = regexp.MustCompile(`(?i)(активность|activity)\s*[:=]?\s*(low|mid|high|низк|сред|высок)`)
	reGoal   = regexp.MustCompile(`(?i)(цель|goal)\s*[:=]?\s*(похуд|сушк|cut|баланс|maint|maintenance|набор|bulk|масса)`)
)

func ParseProfileText(chatID int64, text string) domain.Profile {
	t := strings.TrimSpace(text)

	p := domain.Profile{ChatID: chatID, Activity: "mid", Goal: "balance"}

	if m := reHeight.FindStringSubmatch(t); len(m) >= 3 {
		p.HeightCM, _ = strconv.Atoi(m[2])
	}
	if m := reWeight.FindStringSubmatch(t); len(m) >= 3 {
		p.WeightKG = parseFloat(m[2])
	}
	if m := reBF.FindStringSubmatch(t); len(m) >= 3 {
		p.BodyFatPct = parseFloat(m[2])
	}
	if m := reAge.FindStringSubmatch(t); len(m) >= 3 {
		p.Age, _ = strconv.Atoi(m[2])
	}
	if m := reSex.FindStringSubmatch(t); len(m) >= 3 {
		v := strings.ToLower(m[2])
		switch v {
		case "m", "male", "м":
			p.Sex = "m"
		case "f", "female", "ж":
			p.Sex = "f"
		}
	}
	if m := reAct.FindStringSubmatch(t); len(m) >= 3 {
		v := strings.ToLower(m[2])
		switch {
		case strings.HasPrefix(v, "low") || strings.HasPrefix(v, "низ"):
			p.Activity = "low"
		case strings.HasPrefix(v, "high") || strings.HasPrefix(v, "выс"):
			p.Activity = "high"
		default:
			p.Activity = "mid"
		}
	}
	if m := reGoal.FindStringSubmatch(t); len(m) >= 3 {
		v := strings.ToLower(m[2])
		switch {
		case strings.HasPrefix(v, "похуд") || strings.HasPrefix(v, "суш") || strings.HasPrefix(v, "cut"):
			p.Goal = "cut"
		case strings.HasPrefix(v, "набор") || strings.HasPrefix(v, "bulk") || strings.HasPrefix(v, "мас"):
			p.Goal = "bulk"
		default:
			p.Goal = "balance"
		}
	}
	return p
}

func parseFloat(s string) float64 {
	s = strings.ReplaceAll(s, ",", ".")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
