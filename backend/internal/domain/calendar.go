package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

type Calendar struct {
	ID                  int64     `json:"id"`
	OwnerUserID         int64     `json:"ownerUserId"`
	Name                string    `json:"name"`
	Description         string    `json:"description,omitempty"`
	Color               string    `json:"color"`
	Timezone            string    `json:"timezone"`
	Visible             bool      `json:"visible"`
	ReminderEnabled     bool      `json:"reminderEnabled"`
	ReminderDaysBefore  int       `json:"reminderDaysBefore"`
	ReminderTime        string    `json:"reminderTime"`
	SameDayReminderTime string    `json:"sameDayReminderTime"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type CalendarShare struct {
	ID         int64     `json:"id"`
	CalendarID int64     `json:"calendarId"`
	UserID     int64     `json:"userId"`
	Role       RoleName  `json:"role"`
	CreatedAt  time.Time `json:"createdAt"`
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func ValidateCalendar(calendar Calendar) error {
	if strings.TrimSpace(calendar.Name) == "" || len(calendar.Name) > 120 {
		return errors.New("calendar name is required and must be at most 120 characters")
	}
	if calendar.Color == "" {
		calendar.Color = "#6d8cff"
	}
	if !colorPattern.MatchString(calendar.Color) {
		return errors.New("calendar color must be a hex color")
	}
	if len(calendar.Description) > 1000 {
		return errors.New("calendar description is too long")
	}
	if calendar.ReminderDaysBefore < 0 || calendar.ReminderDaysBefore > 365 {
		return errors.New("calendar reminder days must be between 0 and 365")
	}
	if calendar.ReminderTime == "" {
		calendar.ReminderTime = "09:00"
	}
	if _, err := time.Parse("15:04", calendar.ReminderTime); err != nil {
		return errors.New("calendar reminder time is invalid")
	}
	if calendar.SameDayReminderTime == "" {
		calendar.SameDayReminderTime = "09:00"
	}
	if _, err := time.Parse("15:04", calendar.SameDayReminderTime); err != nil {
		return errors.New("calendar same-day reminder time is invalid")
	}
	return nil
}
