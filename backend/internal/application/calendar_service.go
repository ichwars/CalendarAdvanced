package application

import (
	"fmt"
	"strings"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type CalendarService struct {
	Store *sqlite.Store
	Audit *AuditService
}

type CalendarInput struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	Color               string `json:"color"`
	Timezone            string `json:"timezone"`
	Visible             bool   `json:"visible"`
	ReminderEnabled     bool   `json:"reminderEnabled"`
	ReminderDaysBefore  int    `json:"reminderDaysBefore"`
	ReminderTime        string `json:"reminderTime"`
	SameDayReminderTime string `json:"sameDayReminderTime"`
}

func (s *CalendarService) List(user domain.User) ([]domain.Calendar, error) {
	return s.Store.ListCalendars(user.ID)
}

func (s *CalendarService) Create(input CalendarInput, user domain.User, ip, userAgent string) (domain.Calendar, error) {
	calendar := calendarFromInput(input, user.ID)
	calendar.Visible = true
	if err := domain.ValidateCalendar(calendar); err != nil {
		return domain.Calendar{}, NewError("invalid_calendar", err.Error(), nil)
	}
	created, err := s.Store.CreateCalendar(calendar)
	if err != nil {
		return domain.Calendar{}, err
	}
	s.Audit.Record(user.ID, domain.AuditCalendarChanged, "calendar", fmt.Sprint(created.ID), ip, userAgent, map[string]any{"operation": "create"})
	return created, nil
}

func (s *CalendarService) Update(id int64, input CalendarInput, user domain.User, ip, userAgent string) (domain.Calendar, error) {
	calendar := calendarFromInput(input, user.ID)
	calendar.ID = id
	if err := domain.ValidateCalendar(calendar); err != nil {
		return domain.Calendar{}, NewError("invalid_calendar", err.Error(), nil)
	}
	updated, err := s.Store.UpdateCalendar(calendar, user.ID)
	if err != nil {
		return domain.Calendar{}, err
	}
	s.Audit.Record(user.ID, domain.AuditCalendarChanged, "calendar", fmt.Sprint(updated.ID), ip, userAgent, map[string]any{"operation": "update"})
	return updated, nil
}

func calendarFromInput(input CalendarInput, ownerUserID int64) domain.Calendar {
	reminderTime := input.ReminderTime
	if reminderTime == "" {
		reminderTime = "09:00"
	}
	sameDayReminderTime := input.SameDayReminderTime
	if sameDayReminderTime == "" {
		sameDayReminderTime = "09:00"
	}
	return domain.Calendar{
		OwnerUserID:         ownerUserID,
		Name:                strings.TrimSpace(input.Name),
		Description:         input.Description,
		Color:               input.Color,
		Timezone:            input.Timezone,
		Visible:             input.Visible,
		ReminderEnabled:     input.ReminderEnabled,
		ReminderDaysBefore:  input.ReminderDaysBefore,
		ReminderTime:        reminderTime,
		SameDayReminderTime: sameDayReminderTime,
	}
}

func (s *CalendarService) Delete(id int64, user domain.User, ip, userAgent string) error {
	if err := s.Store.DeleteCalendar(id, user.ID); err != nil {
		return err
	}
	s.Audit.Record(user.ID, domain.AuditCalendarChanged, "calendar", fmt.Sprint(id), ip, userAgent, map[string]any{"operation": "delete"})
	return nil
}
