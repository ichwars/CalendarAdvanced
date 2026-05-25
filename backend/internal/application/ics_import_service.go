package application

import (
	"fmt"
	"strings"
	"time"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/ics"
	"calendaradvanced/internal/infrastructure/sqlite"
)

const maxICSImportBytes = 5 * 1024 * 1024

func MaxICSImportBytes() int64 {
	return maxICSImportBytes
}

type ICSImportService struct {
	Store *sqlite.Store
	Audit *AuditService
}

type ICSImportPreview struct {
	AllDayCount    int               `json:"allDayCount"`
	EventCount     int               `json:"eventCount"`
	RecurringCount int               `json:"recurringCount"`
	RangeEnd       time.Time         `json:"rangeEnd,omitempty"`
	RangeStart     time.Time         `json:"rangeStart,omitempty"`
	Samples        []ICSImportSample `json:"samples"`
	Warnings       []string          `json:"warnings"`
}

type ICSImportSample struct {
	AllDay   bool      `json:"allDay"`
	EndsAt   time.Time `json:"endsAt"`
	Location string    `json:"location,omitempty"`
	StartsAt time.Time `json:"startsAt"`
	Title    string    `json:"title"`
}

type ICSImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Warnings []string `json:"warnings"`
}

func (s *ICSImportService) Preview(data []byte, timezone string) (ICSImportPreview, error) {
	events, warnings, err := s.parse(data, timezone)
	if err != nil {
		return ICSImportPreview{}, NewError("invalid_ics", err.Error(), nil)
	}
	return previewFromEvents(events, warnings), nil
}

func (s *ICSImportService) Import(data []byte, calendarID int64, user domain.User, timezone string, ip, userAgent string) (ICSImportResult, error) {
	if calendarID <= 0 {
		return ICSImportResult{}, NewError("invalid_import", "calendarId is required", nil)
	}
	calendar, err := s.Store.FindCalendarByID(calendarID, user.ID)
	if err != nil {
		return ICSImportResult{}, err
	}
	events, warnings, err := s.parse(data, timezone)
	if err != nil {
		return ICSImportResult{}, NewError("invalid_ics", err.Error(), nil)
	}

	result := ICSImportResult{Warnings: stringSlice(warnings)}
	for _, item := range events {
		eventTimezone := normalizeTimezone(item.Timezone, timezone)
		event := domain.Event{
			CalendarID:  calendarID,
			UID:         item.UID,
			Title:       item.Title,
			Description: item.Description,
			Location:    item.Location,
			StartsAt:    item.StartsAt,
			EndsAt:      item.EndsAt,
			Timezone:    eventTimezone,
			AllDay:      item.AllDay,
			Status:      domain.EventStatusConfirmed,
			CreatedBy:   user.ID,
			Recurrence:  recurrenceFromRRULE(item.RRule),
			Reminders:   remindersForImport(item, eventTimezone, calendar),
		}
		if err := domain.ValidateEvent(event); err != nil {
			result.Skipped++
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", item.Title, err.Error()))
			continue
		}
		if _, err := s.Store.CreateEvent(event); err != nil {
			result.Skipped++
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: konnte nicht importiert werden", item.Title))
			continue
		}
		result.Imported++
	}
	s.Audit.Record(user.ID, domain.AuditEventChanged, "ics_import", fmt.Sprint(calendarID), ip, userAgent, map[string]any{"imported": result.Imported, "skipped": result.Skipped})
	return result, nil
}

func (s *ICSImportService) parse(data []byte, timezone string) ([]ics.Event, []string, error) {
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("file is empty")
	}
	if len(data) > maxICSImportBytes {
		return nil, nil, fmt.Errorf("file is too large")
	}
	parsed, err := ics.ParseCalendar(data, timezone)
	if err != nil {
		return nil, nil, err
	}
	if len(parsed.Events) == 0 {
		return nil, parsed.Warnings, fmt.Errorf("no importable events found")
	}
	return parsed.Events, parsed.Warnings, nil
}

func previewFromEvents(events []ics.Event, warnings []string) ICSImportPreview {
	preview := ICSImportPreview{EventCount: len(events), Samples: []ICSImportSample{}, Warnings: stringSlice(warnings)}
	for index, event := range events {
		if index == 0 || event.StartsAt.Before(preview.RangeStart) {
			preview.RangeStart = event.StartsAt
		}
		if index == 0 || event.EndsAt.After(preview.RangeEnd) {
			preview.RangeEnd = event.EndsAt
		}
		if event.AllDay {
			preview.AllDayCount++
		}
		if event.RRule != "" {
			preview.RecurringCount++
		}
		if len(preview.Samples) < 5 {
			preview.Samples = append(preview.Samples, ICSImportSample{Title: event.Title, Location: event.Location, StartsAt: event.StartsAt, EndsAt: event.EndsAt, AllDay: event.AllDay})
		}
	}
	return preview
}

func stringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func normalizeTimezone(value, fallback string) string {
	if value != "" {
		if _, err := time.LoadLocation(value); err == nil {
			return value
		}
	}
	if fallback != "" {
		return fallback
	}
	return "UTC"
}

func recurrenceFromRRULE(value string) *domain.Recurrence {
	if value == "" {
		return nil
	}
	parts := map[string]string{}
	for _, item := range strings.Split(value, ";") {
		key, val, ok := strings.Cut(item, "=")
		if ok {
			parts[strings.ToUpper(key)] = val
		}
	}
	frequency := domain.RecurrenceFrequency(parts["FREQ"])
	if frequency != domain.FrequencyDaily && frequency != domain.FrequencyWeekly && frequency != domain.FrequencyMonthly && frequency != domain.FrequencyYearly {
		return nil
	}
	recurrence := &domain.Recurrence{Frequency: frequency, Interval: 1, RRule: value, ByDay: parts["BYDAY"]}
	if interval := strings.TrimSpace(parts["INTERVAL"]); interval != "" {
		if parsed, err := parsePositiveInt(interval); err == nil {
			recurrence.Interval = parsed
		}
	}
	if count := strings.TrimSpace(parts["COUNT"]); count != "" {
		if parsed, err := parsePositiveInt(count); err == nil {
			recurrence.Count = parsed
		}
	}
	if until := strings.TrimSpace(parts["UNTIL"]); until != "" {
		if parsed, err := time.Parse("20060102T150405Z", until); err == nil {
			recurrence.Until = parsed
		} else if parsed, err := time.Parse("20060102", until); err == nil {
			recurrence.Until = parsed
		}
	}
	return recurrence
}

func remindersFromMinutes(minutes []int) []domain.Reminder {
	reminders := make([]domain.Reminder, 0, len(minutes))
	for _, item := range minutes {
		if item > 0 {
			reminders = append(reminders, domain.Reminder{MinutesBefore: item})
		}
	}
	return reminders
}

func remindersForImport(event ics.Event, timezone string, calendar domain.Calendar) []domain.Reminder {
	reminders := remindersFromMinutes(event.ReminderMin)
	if !calendar.ReminderEnabled {
		return reminders
	}
	for _, minutesBefore := range calendarReminderMinutes(event.StartsAt, timezone, calendar) {
		if minutesBefore <= 0 {
			continue
		}
		exists := false
		for _, reminder := range reminders {
			if reminder.MinutesBefore == minutesBefore {
				exists = true
				break
			}
		}
		if !exists {
			reminders = append(reminders, domain.Reminder{MinutesBefore: minutesBefore})
		}
	}
	return reminders
}

func calendarReminderMinutes(startsAt time.Time, timezone string, calendar domain.Calendar) []int {
	daysBefore := calendar.ReminderDaysBefore
	if daysBefore < 1 {
		daysBefore = 1
	}
	location, err := time.LoadLocation(normalizeTimezone(timezone, "UTC"))
	if err != nil {
		return nil
	}
	startLocal := startsAt.In(location)
	minutes := []int{}
	if minutesBefore, ok := reminderMinutesAt(startLocal, calendar.ReminderTime, -daysBefore); ok {
		minutes = append(minutes, minutesBefore)
	}
	if minutesBefore, ok := reminderMinutesAt(startLocal, calendar.SameDayReminderTime, 0); ok {
		minutes = append(minutes, minutesBefore)
	}
	return minutes
}

func reminderMinutesAt(startsAt time.Time, clockValue string, dayOffset int) (int, bool) {
	if strings.TrimSpace(clockValue) == "" {
		clockValue = "09:00"
	}
	clock, err := time.Parse("15:04", clockValue)
	if err != nil {
		return 0, false
	}
	reminderAt := time.Date(startsAt.Year(), startsAt.Month(), startsAt.Day(), clock.Hour(), clock.Minute(), 0, 0, startsAt.Location()).AddDate(0, 0, dayOffset)
	duration := startsAt.Sub(reminderAt)
	if duration <= 0 {
		return 0, false
	}
	return int(duration / time.Minute), true
}

func parsePositiveInt(value string) (int, error) {
	var number int
	_, err := fmt.Sscanf(value, "%d", &number)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("invalid number")
	}
	return number, nil
}
