package caldav

import (
	"fmt"
	"strings"
	"time"

	"calendaradvanced/internal/domain"
)

func CalendarData(events []domain.Event) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//CalendarAdvanced//Self Hosted Calendar//EN\r\nCALSCALE:GREGORIAN\r\n")
	for _, event := range events {
		b.WriteString(EventData(event))
	}
	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}

func EventData(event domain.Event) string {
	uid := event.UID
	if uid == "" {
		uid = fmt.Sprintf("event-%d@calendaradvanced", event.ID)
	}
	var b strings.Builder
	b.WriteString("BEGIN:VEVENT\r\n")
	line(&b, "UID", uid)
	line(&b, "SUMMARY", event.Title)
	line(&b, "DESCRIPTION", event.Description)
	line(&b, "LOCATION", event.Location)
	line(&b, "DTSTAMP", time.Now().UTC().Format("20060102T150405Z"))
	line(&b, "DTSTART", event.StartsAt.UTC().Format("20060102T150405Z"))
	line(&b, "DTEND", event.EndsAt.UTC().Format("20060102T150405Z"))
	if event.Recurrence != nil && event.Recurrence.RRule != "" {
		line(&b, "RRULE", event.Recurrence.RRule)
	}
	b.WriteString("END:VEVENT\r\n")
	return b.String()
}

func line(b *strings.Builder, key, value string) {
	if value == "" {
		return
	}
	b.WriteString(key)
	b.WriteString(":")
	b.WriteString(escape(value))
	b.WriteString("\r\n")
}

func escape(v string) string {
	v = strings.ReplaceAll(v, "\\", "\\\\")
	v = strings.ReplaceAll(v, ";", "\\;")
	v = strings.ReplaceAll(v, ",", "\\,")
	v = strings.ReplaceAll(v, "\n", "\\n")
	v = strings.ReplaceAll(v, "\r", "")
	return v
}
