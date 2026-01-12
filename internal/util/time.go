package util

import (
	"time"
)

func MustLocation(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

func NowIn(loc *time.Location) time.Time {
	return time.Now().In(loc)
}

// Monday=1 ... Sunday=7
func Weekday1to7(t time.Time) int {
	wd := int(t.Weekday()) // Sunday=0 ... Saturday=6
	if wd == 0 {
		return 7
	}
	return wd
}

func DayStart(t time.Time, loc *time.Location) time.Time {
	tt := t.In(loc)
	return time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, loc)
}

func LocalDateStr(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("2006-01-02")
}
