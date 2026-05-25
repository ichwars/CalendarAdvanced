package application

import (
	"fmt"
	"time"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type EventService struct {
	Store      *sqlite.Store
	Audit      *AuditService
	Recurrence *RecurrenceService
}

type EventInput struct {
	CalendarID   int64              `json:"calendarId"`
	Title        string             `json:"title"`
	Description  string             `json:"description"`
	Location     string             `json:"location"`
	StartsAt     time.Time          `json:"startsAt"`
	EndsAt       time.Time          `json:"endsAt"`
	Timezone     string             `json:"timezone"`
	AllDay       bool               `json:"allDay"`
	Private      bool               `json:"private"`
	Completed    bool               `json:"completed"`
	BirthdayYear int                `json:"birthdayYear"`
	Recurrence   *domain.Recurrence `json:"recurrence"`
	Attendees    []domain.Attendee  `json:"attendees"`
	Reminders    []domain.Reminder  `json:"reminders"`
}

type EventListInput struct {
	From       time.Time
	To         time.Time
	CalendarID int64
	Query      string
	Limit      int
	Offset     int
	Expand     bool
}

func (s *EventService) List(user domain.User, input EventListInput) ([]domain.Event, error) {
	events, err := s.Store.ListEvents(sqlite.EventFilter{UserID: user.ID, From: input.From, To: input.To, CalendarID: input.CalendarID, Query: input.Query, Limit: input.Limit, Offset: input.Offset, IncludeRecurring: input.Expand})
	if err != nil {
		return nil, err
	}
	annotateDAVSyncedEvents(events, s.Store, user.ID)
	if !input.Expand || input.From.IsZero() || input.To.IsZero() {
		return events, nil
	}
	expanded := make([]domain.Event, 0, len(events))
	for _, event := range events {
		expanded = append(expanded, s.Recurrence.Expand(event, input.From, input.To, input.Limit)...)
	}
	annotateDAVSyncedEvents(expanded, s.Store, user.ID)
	return expanded, nil
}

func (s *EventService) Create(input EventInput, user domain.User, ip, userAgent string) (domain.Event, error) {
	event := eventFromInput(input)
	event.CreatedBy = user.ID
	if err := domain.ValidateEvent(event); err != nil {
		return domain.Event{}, NewError("invalid_event", err.Error(), nil)
	}
	if event.Recurrence != nil {
		if err := domain.ValidateRecurrence(*event.Recurrence); err != nil {
			return domain.Event{}, NewError("invalid_recurrence", err.Error(), nil)
		}
	}
	created, err := s.Store.CreateEvent(event)
	if err != nil {
		return domain.Event{}, err
	}
	conflicts, _ := s.Store.FindConflicts(user.ID, created)
	created.Conflicts = conflicts
	s.Audit.Record(user.ID, domain.AuditEventChanged, "event", fmt.Sprint(created.ID), ip, userAgent, map[string]any{"operation": "create"})
	return created, nil
}

func (s *EventService) Update(id int64, input EventInput, user domain.User, ip, userAgent string) (domain.Event, error) {
	event := eventFromInput(input)
	event.ID = id
	event.CreatedBy = user.ID
	if err := domain.ValidateEvent(event); err != nil {
		return domain.Event{}, NewError("invalid_event", err.Error(), nil)
	}
	if event.Recurrence != nil {
		if err := domain.ValidateRecurrence(*event.Recurrence); err != nil {
			return domain.Event{}, NewError("invalid_recurrence", err.Error(), nil)
		}
	}
	updated, err := s.Store.UpdateEvent(event, user.ID)
	if err != nil {
		return domain.Event{}, err
	}
	conflicts, _ := s.Store.FindConflicts(user.ID, updated)
	updated.Conflicts = conflicts
	s.Audit.Record(user.ID, domain.AuditEventChanged, "event", fmt.Sprint(updated.ID), ip, userAgent, map[string]any{"operation": "update"})
	return updated, nil
}

func (s *EventService) Delete(id int64, user domain.User, ip, userAgent string) error {
	if err := s.Store.DeleteEvent(id, user.ID); err != nil {
		return err
	}
	s.Audit.Record(user.ID, domain.AuditEventChanged, "event", fmt.Sprint(id), ip, userAgent, map[string]any{"operation": "delete"})
	return nil
}

func eventFromInput(input EventInput) domain.Event {
	return domain.Event{CalendarID: input.CalendarID, Title: input.Title, Description: input.Description, Location: input.Location, StartsAt: input.StartsAt, EndsAt: input.EndsAt, Timezone: input.Timezone, AllDay: input.AllDay, Private: input.Private, Completed: input.Completed, BirthdayYear: input.BirthdayYear, Status: domain.EventStatusConfirmed, Recurrence: input.Recurrence, Attendees: input.Attendees, Reminders: input.Reminders}
}

func annotateDAVSyncedEvents(events []domain.Event, store *sqlite.Store, userID int64) {
	synced, err := store.ListDAVSyncedLocalIDs(userID, "event")
	if err != nil {
		return
	}
	for i := range events {
		events[i].DAVSynced = synced[events[i].ID]
	}
}
