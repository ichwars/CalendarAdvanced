package application

import (
	"time"

	"calendaradvanced/internal/domain"
)

type RecurrenceService struct{}

func (s *RecurrenceService) Expand(event domain.Event, from, to time.Time, max int) []domain.Event {
	if event.Recurrence == nil || event.Recurrence.Frequency == "" {
		if event.EndsAt.After(from) && event.StartsAt.Before(to) {
			return []domain.Event{event}
		}
		return nil
	}
	if max <= 0 || max > 1000 {
		max = 250
	}
	out := make([]domain.Event, 0)
	currentStart := event.StartsAt
	duration := event.EndsAt.Sub(event.StartsAt)
	interval := event.Recurrence.Interval
	if interval <= 0 {
		interval = 1
	}
	for occurrence := 0; occurrence < max; occurrence++ {
		currentEnd := currentStart.Add(duration)
		if event.Recurrence.Count > 0 && occurrence >= event.Recurrence.Count {
			break
		}
		if !event.Recurrence.Until.IsZero() && currentStart.After(event.Recurrence.Until) {
			break
		}
		if currentEnd.After(from) && currentStart.Before(to) {
			copy := event
			copy.StartsAt = currentStart
			copy.EndsAt = currentEnd
			out = append(out, copy)
		}
		if currentStart.After(to) {
			break
		}
		switch event.Recurrence.Frequency {
		case domain.FrequencyDaily:
			currentStart = currentStart.AddDate(0, 0, interval)
		case domain.FrequencyWeekly:
			currentStart = currentStart.AddDate(0, 0, 7*interval)
		case domain.FrequencyMonthly:
			currentStart = currentStart.AddDate(0, interval, 0)
		case domain.FrequencyYearly:
			currentStart = currentStart.AddDate(interval, 0, 0)
		default:
			return out
		}
	}
	return out
}
