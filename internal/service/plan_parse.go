package service

import (
	"regexp"
	"strings"
)

var dayHeaderRe = regexp.MustCompile(`(?m)^\s*([1-7])\s*$`)

// SplitPlanByDays режет план по строкам, где строка — это "1".."7".
// Возвращает map[day]text (текст дня без заголовка).
func SplitPlanByDays(plan string) map[int]string {
	plan = strings.ReplaceAll(plan, "\r\n", "\n")

	matches := dayHeaderRe.FindAllStringSubmatchIndex(plan, -1)
	res := make(map[int]string)
	if len(matches) == 0 {
		return res
	}

	for i := 0; i < len(matches); i++ {
		// match: [fullStart fullEnd group1Start group1End]
		g1s, g1e := matches[i][2], matches[i][3]
		dayStr := strings.TrimSpace(plan[g1s:g1e])
		day := int(dayStr[0] - '0')

		start := matches[i][1] // конец строки с номером дня
		end := len(plan)
		if i+1 < len(matches) {
			end = matches[i+1][0] // начало следующего заголовка
		}

		block := strings.TrimSpace(plan[start:end])
		res[day] = block
	}

	return res
}
