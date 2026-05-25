package domain

import (
	"errors"
	"strings"
	"time"
)

type EventStatus string

const (
	EventStatusConfirmed EventStatus = "confirmed"
	EventStatusCancelled EventStatus = "cancelled"
)

type AttendeeStatus string

const (
	AttendeeNeedsAction AttendeeStatus = "needs_action"
	AttendeeAccepted    AttendeeStatus = "accepted"
	AttendeeDeclined    AttendeeStatus = "declined"
	AttendeeTentative   AttendeeStatus = "tentative"
)

type Event struct {
	ID           int64           `json:"id"`
	CalendarID   int64           `json:"calendarId"`
	UID          string          `json:"uid"`
	Title        string          `json:"title"`
	Description  string          `json:"description,omitempty"`
	Location     string          `json:"location,omitempty"`
	StartsAt     time.Time       `json:"startsAt"`
	EndsAt       time.Time       `json:"endsAt"`
	Timezone     string          `json:"timezone"`
	AllDay       bool            `json:"allDay"`
	Private      bool            `json:"private"`
	Completed    bool            `json:"completed"`
	BirthdayYear int             `json:"birthdayYear,omitempty"`
	Status       EventStatus     `json:"status"`
	ETag         string          `json:"etag"`
	CreatedBy    int64           `json:"createdBy"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	Recurrence   *Recurrence     `json:"recurrence,omitempty"`
	Attendees    []Attendee      `json:"attendees,omitempty"`
	Reminders    []Reminder      `json:"reminders,omitempty"`
	Conflicts    []EventConflict `json:"conflicts,omitempty"`
	DAVSynced    bool            `json:"davSynced,omitempty"`
}

type EventConflict struct {
	EventID int64  `json:"eventId"`
	Title   string `json:"title"`
}

type Attendee struct {
	ID          int64          `json:"id"`
	EventID     int64          `json:"eventId"`
	Email       string         `json:"email"`
	DisplayName string         `json:"displayName,omitempty"`
	Status      AttendeeStatus `json:"status"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type Reminder struct {
	ID            int64     `json:"id"`
	EventID       int64     `json:"eventId"`
	MinutesBefore int       `json:"minutesBefore"`
	DeliveredAt   time.Time `json:"deliveredAt,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type DueReminder struct {
	ID            int64     `json:"id"`
	EventID       int64     `json:"eventId"`
	TaskID        int64     `json:"taskId,omitempty"`
	Kind          string    `json:"kind"`
	Title         string    `json:"title"`
	CalendarName  string    `json:"calendarName"`
	StartsAt      time.Time `json:"startsAt"`
	MinutesBefore int       `json:"minutesBefore"`
	DueAt         time.Time `json:"dueAt"`
}

func ValidateEvent(event Event) error {
	if event.CalendarID <= 0 {
		return errors.New("calendar id is required")
	}
	if strings.TrimSpace(event.Title) == "" || len(event.Title) > 200 {
		return errors.New("event title is required and must be at most 200 characters")
	}
	if len(event.Description) > 5000 {
		return errors.New("event description is too long")
	}
	if len(event.Location) > 500 {
		return errors.New("event location is too long")
	}
	if event.StartsAt.IsZero() || event.EndsAt.IsZero() || !event.EndsAt.After(event.StartsAt) {
		return errors.New("event end must be after start")
	}
	if event.BirthdayYear != 0 && (event.BirthdayYear < 1850 || event.BirthdayYear > event.StartsAt.Year()) {
		return errors.New("birthday year is invalid")
	}
	if event.Timezone == "" {
		return errors.New("event timezone is required")
	}
	if _, err := time.LoadLocation(event.Timezone); err != nil {
		return errors.New("event timezone is invalid")
	}
	if event.Status == "" {
		event.Status = EventStatusConfirmed
	}
	return nil
}
