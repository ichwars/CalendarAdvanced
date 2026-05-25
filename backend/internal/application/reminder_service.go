package application

import (
	"time"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type ReminderService struct {
	Store *sqlite.Store
}

func (s *ReminderService) Due(user domain.User, now time.Time) ([]domain.DueReminder, error) {
	return s.Store.ListDueReminders(user.ID, now.UTC(), 50)
}

func (s *ReminderService) MarkDelivered(user domain.User, id int64) error {
	if id <= 0 {
		return ErrValidation
	}
	return s.Store.MarkReminderDelivered(id, user.ID, time.Now().UTC())
}
