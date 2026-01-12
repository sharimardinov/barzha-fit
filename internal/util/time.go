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
