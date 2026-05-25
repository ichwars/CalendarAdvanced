package sqlite

import (
	"path/filepath"
	"testing"
	"time"

	"calendaradvanced/internal/domain"
)

func TestStoreMigrationsSessionsEventsAndCalDAV(t *testing.T) {
	store, err := OpenStore(t.TempDir(), filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	required, err := store.SetupRequired()
	if err != nil {
		t.Fatal(err)
	}
	if !required {
		t.Fatal("fresh store should require setup")
	}
	user, err := store.CreateUser("admin@example.test", "admin", "Admin", "$argon2id$placeholder", true, []domain.RoleName{domain.RoleAdmin, domain.RoleEditor})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := store.CreateSession(user.ID, "session-hash", "csrf-hash", time.Now().Add(time.Hour), "agent", "127.0.0.1"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	session, loadedUser, err := store.FindSession("session-hash")
	if err != nil {
		t.Fatalf("find session: %v", err)
	}
	if session.UserID != user.ID || loadedUser.Email != user.Email {
		t.Fatal("session user mismatch")
	}
	calendar, err := store.CreateCalendar(domain.Calendar{OwnerUserID: user.ID, Name: "Private", Color: "#6d8cff", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	event, err := store.CreateEvent(domain.Event{CalendarID: calendar.ID, Title: "Meeting", StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now().Add(2 * time.Hour), Timezone: "UTC", CreatedBy: user.ID, BirthdayYear: 1990, Recurrence: &domain.Recurrence{Frequency: domain.FrequencyWeekly, Interval: 1, Count: 2}})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	if event.BirthdayYear != 1990 {
		t.Fatalf("birthday year not stored: %d", event.BirthdayYear)
	}
	if event.Recurrence == nil || event.Recurrence.RRule == "" {
		t.Fatal("recurrence not stored")
	}
	if _, err := store.CreateCalDAVToken(user.ID, "DAVx5", "token-hash", "n-hash"); err != nil {
		t.Fatalf("create caldav token: %v", err)
	}
	if _, err := store.FindCalDAVUser(user.Email, "token-hash"); err != nil {
		t.Fatalf("find caldav user: %v", err)
	}
	preferences := domain.GeneralPreferences{
		CalendarDensity:          "compact",
		CompactMode:              true,
		DateFormat:               "de",
		DefaultCalendarView:      "week",
		DefaultEventDuration:     "45",
		HighlightedHolidayRegion: "DE-BY",
		HolidayRegion:            "DE-NW",
		Locale:                   "de",
		RememberLastRoute:        true,
		ShowHolidays:             true,
		ShowWeekends:             false,
		StartPage:                "calendar",
		TimeFormat24h:            true,
		TimeGrid:                 "15",
		Theme:                    "dark",
		Timezone:                 "Europe/Berlin",
		WeekStart:                "monday",
		WorkingHoursEnd:          "16:00",
		WorkingHoursStart:        "07:00",
	}
	if err := store.UpsertUserPreferences(user.ID, preferences); err != nil {
		t.Fatalf("save preferences: %v", err)
	}
	loadedPreferences, err := store.GetUserPreferences(user.ID)
	if err != nil {
		t.Fatalf("load preferences: %v", err)
	}
	if loadedPreferences.CalendarDensity != preferences.CalendarDensity || loadedPreferences.WorkingHoursStart != preferences.WorkingHoursStart {
		t.Fatal("preferences mismatch")
	}
}
