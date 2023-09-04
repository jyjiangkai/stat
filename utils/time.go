package utils

import (
	"time"
)

func GetHourBeginTime(t time.Time) time.Time {
	year, month, day := t.Date()
	endOfDay := time.Date(year, month, day, 23, 59, 59, 999, t.Location())
	return endOfDay
}

// GetDayEndTime return current day end time
func GetDayEndTime(t time.Time) time.Time {
	year, month, day := t.Date()
	endOfDay := time.Date(year, month, day, 23, 59, 59, 999, t.Location())
	return endOfDay
}

func GetDayBeginTime(t time.Time) time.Time {
	year, month, day := t.Date()
	beginOfBegin := time.Date(year, month, day, 0, 0, 0, 0, t.Location())
	return beginOfBegin
}

// GetMonthEndTime return current month end time
func GetMonthEndTime(t time.Time) time.Time {
	return GetMonthBeginTime(t).AddDate(0, 1, 0).Add(-time.Millisecond)
}

func GetMonthBeginTime(t time.Time) time.Time {
	year, month, _ := t.Date()
	endOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
	return endOfMonth
}

func ToBillTimeForConnect(t time.Time) time.Time {
	return t.Add(-24 * time.Hour)
}

func ToBillTimeForAI(t time.Time) time.Time {
	var real time.Time
	if t.Hour() == 0 {
		real = t.Add(-24 * time.Hour)
	} else {
		real = t
	}
	return time.Date(real.Year(), real.Month(), real.Day(), 0, 0, 0, 0, time.UTC)
}
