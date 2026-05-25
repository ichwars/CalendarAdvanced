package domain

type GeneralPreferences struct {
	CalendarDensity          string `json:"calendarDensity"`
	CompactMode              bool   `json:"compactMode"`
	DateFormat               string `json:"dateFormat"`
	DefaultCalendarView      string `json:"defaultCalendarView"`
	DefaultEventDuration     string `json:"defaultEventDuration"`
	HighlightedHolidayRegion string `json:"highlightedHolidayRegion"`
	HolidayRegion            string `json:"holidayRegion"`
	Locale                   string `json:"locale"`
	RememberLastRoute        bool   `json:"rememberLastRoute"`
	ShowHolidays             bool   `json:"showHolidays"`
	ShowWeekends             bool   `json:"showWeekends"`
	StartPage                string `json:"startPage"`
	TimeFormat24h            bool   `json:"timeFormat24h"`
	TimeGrid                 string `json:"timeGrid"`
	Theme                    string `json:"theme"`
	Timezone                 string `json:"timezone"`
	WeekStart                string `json:"weekStart"`
	WorkingHoursEnd          string `json:"workingHoursEnd"`
	WorkingHoursStart        string `json:"workingHoursStart"`
}
