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
