package domain

import (
	"testing"
	"time"
)

func TestPasswordPolicy(t *testing.T) {
	if ValidatePassword("too-short") == nil {
		t.Fatal("weak password accepted")
	}
	if err := ValidatePassword("Strong-Password-123!"); err != nil {
		t.Fatalf("strong password rejected: %v", err)
	}
}

func TestRoles(t *testing.T) {
	roles := []RoleName{RoleViewer, RoleEditor}
	if !HasRole(roles, RoleEditor) {
		t.Fatal("expected editor role")
	}
	if HasRole(roles, RoleAdmin) {
		t.Fatal("unexpected admin role")
	}
}

func TestEventValidationRejectsInvalidTime(t *testing.T) {
	now := time.Now().UTC()
	event := Event{CalendarID: 1, Title: "Test", StartsAt: now, EndsAt: now.Add(-time.Hour), Timezone: "UTC"}
	if ValidateEvent(event) == nil {
		t.Fatal("invalid event time accepted")
	}
}

func TestRecurrenceValidation(t *testing.T) {
	if err := ValidateRecurrence(Recurrence{Frequency: FrequencyWeekly, Interval: 1, Count: 10}); err != nil {
		t.Fatalf("valid recurrence rejected: %v", err)
	}
	if err := ValidateRecurrence(Recurrence{Frequency: FrequencyYearly, Interval: 1, Count: 2}); err != nil {
		t.Fatalf("valid yearly recurrence rejected: %v", err)
	}
	if ValidateRecurrence(Recurrence{Frequency: "HOURLY", Interval: 1}) == nil {
		t.Fatal("unsupported recurrence accepted")
	}
}
