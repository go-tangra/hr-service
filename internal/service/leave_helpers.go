package service

import (
	"time"
)

// calculateBusinessDays calculates the number of business days (weekdays) between two dates, inclusive
func calculateBusinessDays(start, end time.Time) float64 {
	// Normalize to start of day
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())

	if end.Before(start) {
		return 0
	}

	days := 0.0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		wd := d.Weekday()
		if wd != time.Saturday && wd != time.Sunday {
			days++
		}
	}

	return days
}
