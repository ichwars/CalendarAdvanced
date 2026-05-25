package ics

import "testing"

func TestParseAllDayEventWithSameStartAndEndDate(t *testing.T) {
	data := []byte(`BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:test-1
SUMMARY:Biotonne
LOCATION:Bürgerstraße 1
DTSTART;VALUE=DATE:20260527
DTEND;VALUE=DATE:20260527
BEGIN:VALARM
TRIGGER:-PT8H
ACTION:DISPLAY
DESCRIPTION:Biotonne
END:VALARM
END:VEVENT
END:VCALENDAR`)

	result, err := ParseCalendar(data, "Europe/Berlin")
	if err != nil {
		t.Fatalf("parse calendar: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(result.Events))
	}
	event := result.Events[0]
	if !event.AllDay {
		t.Fatal("event should be all day")
	}
	if event.Title != "Biotonne" || event.Location != "Bürgerstraße 1" {
		t.Fatalf("unexpected text fields: %#v", event)
	}
	if !event.EndsAt.After(event.StartsAt) {
		t.Fatal("end should be after start")
	}
	if got := event.EndsAt.Sub(event.StartsAt).Hours(); got != 24 {
		t.Fatalf("duration hours = %.0f, want 24", got)
	}
	if len(event.ReminderMin) != 1 || event.ReminderMin[0] != 480 {
		t.Fatalf("reminder = %#v, want 480 minutes", event.ReminderMin)
	}
}
