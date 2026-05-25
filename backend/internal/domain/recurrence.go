package domain

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

type RecurrenceFrequency string

const (
	FrequencyDaily   RecurrenceFrequency = "DAILY"
	FrequencyWeekly  RecurrenceFrequency = "WEEKLY"
	FrequencyMonthly RecurrenceFrequency = "MONTHLY"
	FrequencyYearly  RecurrenceFrequency = "YEARLY"
)

type Recurrence struct {
	ID        int64               `json:"id"`
	EventID   int64               `json:"eventId"`
	Frequency RecurrenceFrequency `json:"frequency"`
	Interval  int                 `json:"interval"`
	Count     int                 `json:"count,omitempty"`
	Until     time.Time           `json:"until,omitempty"`
	ByDay     string              `json:"byDay,omitempty"`
	RRule     string              `json:"rrule"`
	CreatedAt time.Time           `json:"createdAt"`
}

func ValidateRecurrence(r Recurrence) error {
	if r.Frequency == "" {
		return nil
	}
	switch r.Frequency {
	case FrequencyDaily, FrequencyWeekly, FrequencyMonthly, FrequencyYearly:
	default:
		return errors.New("unsupported recurrence frequency")
	}
	if r.Interval <= 0 || r.Interval > 365 {
		return errors.New("recurrence interval is invalid")
	}
	if r.Count < 0 || r.Count > 1000 {
		return errors.New("recurrence count is invalid")
	}
	if len(r.ByDay) > 80 || strings.ContainsAny(r.ByDay, "\n\r") {
		return errors.New("recurrence byDay is invalid")
	}
	return nil
}

func RRULE(r Recurrence) string {
	if r.Frequency == "" {
		return ""
	}
	parts := []string{"FREQ=" + string(r.Frequency)}
	interval := r.Interval
	if interval <= 0 {
		interval = 1
	}
	parts = append(parts, "INTERVAL="+strconv.Itoa(interval))
	if r.Count > 0 {
		parts = append(parts, "COUNT="+strconv.Itoa(r.Count))
	}
	if !r.Until.IsZero() {
		parts = append(parts, "UNTIL="+r.Until.UTC().Format("20060102T150405Z"))
	}
	if r.ByDay != "" {
		parts = append(parts, "BYDAY="+r.ByDay)
	}
	return strings.Join(parts, ";")
}
