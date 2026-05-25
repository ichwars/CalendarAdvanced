package application

import (
	"regexp"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type PreferencesService struct {
	Store *sqlite.Store
}

type PreferencesResult struct {
	Persisted   bool                      `json:"persisted"`
	Preferences domain.GeneralPreferences `json:"preferences"`
}

var timeValuePattern = regexp.MustCompile(`^\d{2}:\d{2}$`)

func DefaultGeneralPreferences() domain.GeneralPreferences {
	return domain.GeneralPreferences{
		CalendarDensity:          "comfortable",
		CompactMode:              false,
		DateFormat:               "de",
		DefaultCalendarView:      "week",
		DefaultEventDuration:     "60",
		HighlightedHolidayRegion: "",
		HolidayRegion:            "DE",
		Locale:                   "de",
		RememberLastRoute:        true,
		ShowHolidays:             true,
		ShowWeekends:             true,
		StartPage:                "overview",
		TimeFormat24h:            true,
		TimeGrid:                 "15",
		Theme:                    "dark",
		Timezone:                 "Europe/Berlin",
		WeekStart:                "monday",
		WorkingHoursEnd:          "17:00",
		WorkingHoursStart:        "08:00",
	}
}

func (s *PreferencesService) Get(user domain.User) (PreferencesResult, error) {
	preferences, err := s.Store.GetUserPreferences(user.ID)
	if err != nil {
		if err == sqlite.ErrNotFound {
			return PreferencesResult{Persisted: false, Preferences: DefaultGeneralPreferences()}, nil
		}
		return PreferencesResult{}, err
	}
	preferences, err = normalizePreferences(preferences)
	if err != nil {
		return PreferencesResult{}, err
	}
	return PreferencesResult{Persisted: true, Preferences: preferences}, nil
}

func (s *PreferencesService) Save(user domain.User, preferences domain.GeneralPreferences) (PreferencesResult, error) {
	normalized, err := normalizePreferences(preferences)
	if err != nil {
		return PreferencesResult{}, err
	}
	if err := s.Store.UpsertUserPreferences(user.ID, normalized); err != nil {
		return PreferencesResult{}, err
	}
	return PreferencesResult{Persisted: true, Preferences: normalized}, nil
}

func normalizePreferences(input domain.GeneralPreferences) (domain.GeneralPreferences, error) {
	defaults := DefaultGeneralPreferences()
	if input.CalendarDensity == "" {
		input.CalendarDensity = defaults.CalendarDensity
	}
	if input.DateFormat == "" {
		input.DateFormat = defaults.DateFormat
	}
	if input.DefaultCalendarView == "" {
		input.DefaultCalendarView = defaults.DefaultCalendarView
	}
	if input.DefaultEventDuration == "" {
		input.DefaultEventDuration = defaults.DefaultEventDuration
	}
	if input.HolidayRegion == "" && input.HighlightedHolidayRegion == "" && !input.ShowHolidays {
		input.ShowHolidays = defaults.ShowHolidays
	}
	if input.HolidayRegion == "" {
		input.HolidayRegion = defaults.HolidayRegion
	}
	if input.Locale == "" {
		input.Locale = defaults.Locale
	}
	if input.StartPage == "" {
		input.StartPage = defaults.StartPage
	}
	if input.TimeGrid == "" {
		input.TimeGrid = defaults.TimeGrid
	}
	if input.Theme == "" {
		input.Theme = defaults.Theme
	}
	if input.Timezone == "" {
		input.Timezone = defaults.Timezone
	}
	if input.WeekStart == "" {
		input.WeekStart = defaults.WeekStart
	}
	if input.WorkingHoursStart == "" {
		input.WorkingHoursStart = defaults.WorkingHoursStart
	}
	if input.WorkingHoursEnd == "" {
		input.WorkingHoursEnd = defaults.WorkingHoursEnd
	}
	if !oneOf(input.CalendarDensity, "comfortable", "compact", "dense") {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "calendarDensity is invalid", nil)
	}
	if !oneOf(input.DateFormat, "de", "iso") {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "dateFormat is invalid", nil)
	}
	if !oneOf(input.DefaultCalendarView, "day", "week", "month", "agenda") {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "defaultCalendarView is invalid", nil)
	}
	if !oneOf(input.DefaultEventDuration, "30", "45", "60", "90") {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "defaultEventDuration is invalid", nil)
	}
	if !oneOf(input.HolidayRegion, "DE", "DE-BB", "DE-BE", "DE-BW", "DE-BY", "DE-HB", "DE-HE", "DE-HH", "DE-MV", "DE-NI", "DE-NW", "DE-RP", "DE-SH", "DE-SL", "DE-SN", "DE-ST", "DE-TH") {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "holidayRegion is invalid", nil)
	}
	if input.HighlightedHolidayRegion != "" && !oneOf(input.HighlightedHolidayRegion, "DE-BB", "DE-BE", "DE-BW", "DE-BY", "DE-HB", "DE-HE", "DE-HH", "DE-MV", "DE-NI", "DE-NW", "DE-RP", "DE-SH", "DE-SL", "DE-SN", "DE-ST", "DE-TH") {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "highlightedHolidayRegion is invalid", nil)
	}
	if !oneOf(input.Locale, "de", "en") {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "locale is invalid", nil)
	}
	if !oneOf(input.StartPage, "overview", "calendar", "events") {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "startPage is invalid", nil)
	}
	if !oneOf(input.TimeGrid, "5", "10", "15", "30") {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "timeGrid is invalid", nil)
	}
	if !oneOf(input.Theme, "dark", "light", "system") {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "theme is invalid", nil)
	}
	if !oneOf(input.WeekStart, "monday", "sunday") {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "weekStart is invalid", nil)
	}
	if !timeValuePattern.MatchString(input.WorkingHoursStart) || !timeValuePattern.MatchString(input.WorkingHoursEnd) {
		return domain.GeneralPreferences{}, NewError("invalid_preferences", "working hours must use HH:MM", nil)
	}
	return input, nil
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
