package util

import (
	"regexp"
	"strconv"
	"strings"
)

var kcalRe = regexp.MustCompile(`(?i)(\d{1,5})\s*(ккал|kcal|cal)\b`)

func ExtractCalories(text string) int {
	m := kcalRe.FindAllStringSubmatch(text, -1)
	sum := 0
	for _, g := range m {
		n, _ := strconv.Atoi(g[1])
		sum += n
	}
	return sum
}

func NormalizeMealText(s string) string {
	s = strings.TrimSpace(s)
	return s
}
