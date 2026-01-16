package input

import (
	"strconv"
	"strings"
)

func ParseIntInRange(s string, min, max int) (int, bool) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v < min || v > max {
		return 0, false
	}
	return v, true
}

func ParseFloatInRange(s string, min, max float64) (float64, bool) {
	clean := strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil || v < min || v > max {
		return 0, false
	}
	return v, true
}

func ParseSteps(s string) (int, bool) {
	return ParseIntInRange(s, 0, 100000)
}

func ParseWeight(s string) (float64, bool) {
	return ParseFloatInRange(s, 20, 400)
}

func ParseMonthArg(s string) (month int, year int, ok bool) {
	if len(s) != 4 {
		return 0, 0, false
	}
	mm, ok := parseIntStrict(s[:2])
	if !ok || mm < 1 || mm > 12 {
		return 0, 0, false
	}
	yy, ok := parseIntStrict(s[2:])
	if !ok {
		return 0, 0, false
	}
	return mm, 2000 + yy, true
}

func parseIntStrict(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}
