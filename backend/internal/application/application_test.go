package application

import (
	"path/filepath"
	"testing"
	"time"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/config"
	"calendaradvanced/internal/infrastructure/crypto"
	"calendaradvanced/internal/infrastructure/sqlite"
)

func testServices(t *testing.T) *Services {
	t.Helper()
	store, err := sqlite.OpenStore(t.TempDir(), filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	cfg := config.Load("test")
	cfg.SessionTTL = time.Hour
	cfg.LocalResetTokens = true
	cfg.TokenEncryptionKey = "01234567890123456789012345678901"
	return NewServices(cfg, store)
}

func createTestUser(t *testing.T, services *Services) domain.User {
	t.Helper()
	hash, err := crypto.HashPassword("Strong-Password-123!")
	if err != nil {
		t.Fatal(err)
	}
	user, err := services.Store.CreateUser("user@example.test", "example", "Example User", hash, true, []domain.RoleName{domain.RoleAdmin, domain.RoleEditor, domain.RoleViewer})
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimitService()
	if !limiter.Allow("x", 1, time.Minute) {
		t.Fatal("first request should be allowed")
	}
	if limiter.Allow("x", 1, time.Minute) {
		t.Fatal("second request should be denied")
	}
	limiter.Reset("x")
	if !limiter.Allow("x", 1, time.Minute) {
		t.Fatal("request after reset should be allowed")
	}
}

func TestAuthSessionAndCSRF(t *testing.T) {
	services := testServices(t)
	createTestUser(t, services)
	login, err := services.Auth.Login(LoginInput{Email: "user@example.test", Password: "Strong-Password-123!"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	session, _, err := services.Auth.Authenticate(login.SessionToken)
	if err != nil {
		t.Fatalf("authenticate failed: %v", err)
	}
	if err := services.Auth.ValidateCSRF(session, login.CSRFToken); err != nil {
		t.Fatalf("valid csrf rejected: %v", err)
	}
	if services.Auth.ValidateCSRF(session, "wrong") == nil {
		t.Fatal("invalid csrf accepted")
	}
}

func TestTwoFactorEnableAndLogin(t *testing.T) {
	services := testServices(t)
	user := createTestUser(t, services)
	secret, _, err := services.Auth.BeginTwoFactorSetup(user)
	if err != nil {
		t.Fatal(err)
	}
	code, err := crypto.TOTPCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	backupCodes, err := services.Auth.EnableTwoFactor(user, code, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("enable 2fa: %v", err)
	}
	if len(backupCodes) == 0 {
		t.Fatal("backup codes missing")
	}
	_, err = services.Auth.Login(LoginInput{Email: "user@example.test", Password: "Strong-Password-123!"}, "127.0.0.1", "test")
	if err == nil {
		t.Fatal("login without 2fa should fail")
	}
	code, _ = crypto.TOTPCode(secret, time.Now().UTC())
	if _, err := services.Auth.Login(LoginInput{Email: "user@example.test", Password: "Strong-Password-123!", TOTPCode: code}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("login with 2fa failed: %v", err)
	}
}

func TestRecurrenceExpansion(t *testing.T) {
	service := &RecurrenceService{}
	start := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	event := domain.Event{ID: 1, StartsAt: start, EndsAt: start.Add(time.Hour), Recurrence: &domain.Recurrence{Frequency: domain.FrequencyDaily, Interval: 1, Count: 3}}
	items := service.Expand(event, start.Add(-time.Hour), start.AddDate(0, 0, 5), 10)
	if len(items) != 3 {
		t.Fatalf("expected 3 occurrences, got %d", len(items))
	}
}

func TestYearlyRecurrenceExpansion(t *testing.T) {
	service := &RecurrenceService{}
	start := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	event := domain.Event{ID: 1, StartsAt: start, EndsAt: start.Add(time.Hour), Recurrence: &domain.Recurrence{Frequency: domain.FrequencyYearly, Interval: 1, Count: 2}}
	items := service.Expand(event, start.Add(-time.Hour), start.AddDate(3, 0, 0), 10)
	if len(items) != 2 {
		t.Fatalf("expected 2 yearly occurrences, got %d", len(items))
	}
	if items[1].StartsAt.Year() != 2027 {
		t.Fatalf("expected second occurrence in 2027, got %s", items[1].StartsAt)
	}
}

func TestListExpandsYearlyEventFromEarlierRange(t *testing.T) {
	services := testServices(t)
	user := createTestUser(t, services)
	calendar, err := services.Store.CreateCalendar(domain.Calendar{OwnerUserID: user.ID, Name: "Geburtstage", Color: "#d85a8a", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	start := time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC)
	_, err = services.Events.Create(EventInput{
		CalendarID:   calendar.ID,
		Title:        "Daniel Geburtstag",
		StartsAt:     start,
		EndsAt:       start.Add(time.Hour),
		Timezone:     "UTC",
		AllDay:       true,
		BirthdayYear: 1990,
		Recurrence:   &domain.Recurrence{Frequency: domain.FrequencyYearly, Interval: 1},
	}, user, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	items, err := services.Events.List(user, EventListInput{
		From:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		To:     time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
		Expand: true,
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 expanded yearly event, got %d", len(items))
	}
	if items[0].StartsAt.Year() != 2026 || items[0].BirthdayYear != 1990 {
		t.Fatalf("unexpected expanded event: starts=%s birthdayYear=%d", items[0].StartsAt, items[0].BirthdayYear)
	}
}

func TestBackupValidation(t *testing.T) {
	services := testServices(t)
	user := createTestUser(t, services)
	data, err := services.Backup.Export(user, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := services.Backup.ValidateRestore(data); err != nil {
		t.Fatalf("backup validation failed: %v", err)
	}
}
