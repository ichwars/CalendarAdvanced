package application

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/sqlite"
	"calendaradvanced/internal/infrastructure/xlsx"
)

const maxExcelImportBytes = 20 * 1024 * 1024

var sheetYearPattern = regexp.MustCompile(`\b(19|20)\d{2}\b`)
var montageWeekPattern = regexp.MustCompile(`(?i)\bkw\s*(\d{1,2})\b`)

func MaxExcelImportBytes() int64 {
	return maxExcelImportBytes
}

type ExcelImportService struct {
	Store *sqlite.Store
	Audit *AuditService
}

type ExcelImportPreview struct {
	CancelledRows int                 `json:"cancelledRows"`
	EventCount    int                 `json:"eventCount"`
	RangeEnd      time.Time           `json:"rangeEnd,omitempty"`
	RangeStart    time.Time           `json:"rangeStart,omitempty"`
	Rows          int                 `json:"rows"`
	Samples       []ExcelImportSample `json:"samples"`
	Warnings      []string            `json:"warnings"`
}

type ExcelImportSample struct {
	Completed bool      `json:"completed"`
	Date      time.Time `json:"date"`
	Employee  string    `json:"employee,omitempty"`
	Location  string    `json:"location,omitempty"`
	Pop       string    `json:"pop,omitempty"`
	Sheet     string    `json:"sheet"`
	Status    string    `json:"status,omitempty"`
	Title     string    `json:"title"`
	Week      int       `json:"week"`
	Weekday   string    `json:"weekday"`
}

type ExcelImportResult struct {
	Imported         int      `json:"imported"`
	Skipped          int      `json:"skipped"`
	SkippedCancelled int      `json:"skippedCancelled"`
	Updated          int      `json:"updated"`
	Warnings         []string `json:"warnings"`
}

type ExcelImportOptions struct {
	AllDay        bool
	CalendarID    int64
	EmployeeQuery string
	SourceName    string
	Timezone      string
	WorkEnd       string
	WorkStart     string
}

type excelTask struct {
	Date     time.Time
	Employee string
	Location string
	Pop      string
	Row      int
	Sheet    string
	Status   string
	Summary  string
	Week     int
	Weekday  string
}

func (s *ExcelImportService) Preview(data []byte, timezone, employeeQuery, sourceName string) (ExcelImportPreview, error) {
	tasks, warnings, err := parseExcelTasks(data, timezone, sourceName)
	if err != nil {
		return ExcelImportPreview{}, NewError("invalid_excel_import", err.Error(), nil)
	}
	tasks, err = filterExcelTasks(tasks, employeeQuery)
	if err != nil {
		return ExcelImportPreview{}, NewError("invalid_excel_import", err.Error(), nil)
	}
	cancelledRows := countCancelledExcelRows(tasks)
	tasks = filterImportableExcelTasks(tasks)
	preview := excelPreviewFromTasks(tasks, warnings)
	preview.CancelledRows = cancelledRows
	return preview, nil
}

func (s *ExcelImportService) Import(data []byte, options ExcelImportOptions, user domain.User, ip, userAgent string) (ExcelImportResult, error) {
	if options.CalendarID <= 0 {
		return ExcelImportResult{}, NewError("invalid_excel_import", "calendarId is required", nil)
	}
	calendar, err := s.Store.FindCalendarByID(options.CalendarID, user.ID)
	if err != nil {
		return ExcelImportResult{}, err
	}
	tasks, warnings, err := parseExcelTasks(data, options.Timezone, options.SourceName)
	if err != nil {
		return ExcelImportResult{}, NewError("invalid_excel_import", err.Error(), nil)
	}
	tasks, err = filterExcelTasks(tasks, options.EmployeeQuery)
	if err != nil {
		return ExcelImportResult{}, NewError("invalid_excel_import", err.Error(), nil)
	}
	result := ExcelImportResult{Warnings: stringSlice(warnings)}
	for _, task := range tasks {
		if excelTaskCancelled(task) {
			if existing, err := s.Store.FindEventByUID(excelTaskUID(task), user.ID); err == nil && existing.CalendarID == options.CalendarID {
				_ = s.Store.DeleteEvent(existing.ID, user.ID)
			}
			result.Skipped++
			result.SkippedCancelled++
			continue
		}
		startsAt, endsAt := excelTaskTimeRange(task.Date, options)
		event := domain.Event{
			CalendarID:  options.CalendarID,
			UID:         excelTaskUID(task),
			Title:       excelTaskTitle(task),
			Description: excelTaskDescription(task),
			Location:    task.Location,
			StartsAt:    startsAt,
			EndsAt:      endsAt,
			Timezone:    normalizeTimezone("", options.Timezone),
			AllDay:      options.AllDay,
			Status:      domain.EventStatusConfirmed,
			Completed:   excelTaskCompleted(task),
			CreatedBy:   user.ID,
			Reminders:   calendarImportReminders(startsAt, options.Timezone, calendar),
		}
		if err := domain.ValidateEvent(event); err != nil {
			result.Skipped++
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", event.Title, err.Error()))
			continue
		}
		existing, err := s.Store.FindEventByUID(event.UID, user.ID)
		if err == nil {
			if existing.CalendarID != options.CalendarID {
				result.Skipped++
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: existiert bereits in einem anderen Kalender", event.Title))
				continue
			}
			event.ID = existing.ID
			if excelEventsEqual(existing, event) {
				result.Skipped++
				continue
			}
			if _, err := s.Store.UpdateEvent(event, user.ID); err != nil {
				result.Skipped++
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: konnte nicht aktualisiert werden", event.Title))
				continue
			}
			result.Updated++
			continue
		}
		if err != sqlite.ErrNotFound {
			result.Skipped++
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: konnte nicht abgeglichen werden", event.Title))
			continue
		}
		if _, err := s.Store.CreateEvent(event); err != nil {
			result.Skipped++
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: konnte nicht importiert werden", event.Title))
			continue
		}
		result.Imported++
	}
	s.Audit.Record(user.ID, domain.AuditEventChanged, "excel_import", fmt.Sprint(options.CalendarID), ip, userAgent, map[string]any{"imported": result.Imported, "skipped": result.Skipped, "updated": result.Updated})
	return result, nil
}

func filterExcelTasks(tasks []excelTask, employeeQuery string) ([]excelTask, error) {
	employeeQuery = strings.ToLower(strings.TrimSpace(employeeQuery))
	if employeeQuery == "" {
		return tasks, nil
	}
	filtered := make([]excelTask, 0, len(tasks))
	for _, task := range tasks {
		if strings.Contains(strings.ToLower(task.Employee), employeeQuery) {
			filtered = append(filtered, task)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no importable tasks found for employee filter")
	}
	return filtered, nil
}

func parseExcelTasks(data []byte, timezone, sourceName string) ([]excelTask, []string, error) {
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("file is empty")
	}
	if len(data) > maxExcelImportBytes {
		return nil, nil, fmt.Errorf("file is too large")
	}
	workbook, err := xlsx.ParseWorkbook(data)
	if err != nil {
		return nil, nil, err
	}
	location, err := time.LoadLocation(normalizeTimezone("", timezone))
	if err != nil {
		return nil, nil, err
	}
	if tasks, warnings, ok := parseMontagePlanningTasks(workbook, location, sourceName); ok {
		if len(tasks) == 0 {
			return nil, warnings, fmt.Errorf("no importable tasks found")
		}
		return tasks, warnings, nil
	}
	var tasks []excelTask
	warnings := []string{}
	for _, sheet := range workbook.Sheets {
		year, ok := yearFromSheetName(sheet.Name)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("%s: Jahr im Sheet-Namen nicht erkannt", sheet.Name))
			continue
		}
		for _, row := range sheet.Rows {
			week, ok := weekFromCell(cell(row.Cells, 0))
			if !ok {
				continue
			}
			pop := cell(row.Cells, 1)
			locationName := cell(row.Cells, 3)
			employee := cell(row.Cells, 15)
			status := cell(row.Cells, 21)
			if pop == "" && locationName == "" && employee == "" {
				continue
			}
			for offset, column := range []int{16, 17, 18, 19, 20} {
				if !isMarked(cell(row.Cells, column)) {
					continue
				}
				date, err := isoWeekdayDate(year, week, offset, location)
				if err != nil {
					warnings = append(warnings, fmt.Sprintf("%s Zeile %d: KW %d ist ungültig", sheet.Name, row.Index, week))
					continue
				}
				tasks = append(tasks, excelTask{Date: date, Employee: employee, Location: locationName, Pop: pop, Row: row.Index, Sheet: sheet.Name, Status: status, Week: week, Weekday: weekdayName(offset)})
			}
		}
	}
	if len(tasks) == 0 {
		return nil, warnings, fmt.Errorf("no importable tasks found")
	}
	return tasks, warnings, nil
}

func parseMontagePlanningTasks(workbook xlsx.Workbook, location *time.Location, sourceName string) ([]excelTask, []string, bool) {
	var sheet *xlsx.Sheet
	for index := range workbook.Sheets {
		if strings.EqualFold(strings.TrimSpace(workbook.Sheets[index].Name), "PROJALLG") {
			sheet = &workbook.Sheets[index]
			break
		}
	}
	if sheet == nil || !looksLikeMontagePlanningSheet(*sheet) {
		return nil, nil, false
	}
	week, ok := montagePlanningWeek(*sheet, sourceName)
	warnings := []string{}
	if !ok {
		return nil, []string{"PROJALLG: Kalenderwoche nicht erkannt"}, true
	}
	year := montagePlanningYear(week, time.Now().In(location))
	tasks := []excelTask{}
	for _, row := range sheet.Rows {
		if row.Index <= 2 {
			continue
		}
		employee := cell(row.Cells, 0)
		project := cell(row.Cells, 1)
		title := cell(row.Cells, 2)
		if employee == "" && project == "" && title == "" {
			continue
		}
		for offset, column := range []int{3, 4, 5, 6, 7, 8, 9} {
			marker := strings.TrimSpace(cell(row.Cells, column))
			if marker == "" {
				continue
			}
			date, err := isoWeekdayDate(year, week, offset, location)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("PROJALLG Zeile %d: KW %d ist ungültig", row.Index, week))
				continue
			}
			tasks = append(tasks, excelTask{
				Date:     date,
				Employee: employee,
				Pop:      project,
				Row:      row.Index,
				Sheet:    sheet.Name,
				Status:   "open",
				Summary:  title,
				Week:     week,
				Weekday:  weekdayName(offset),
			})
		}
	}
	return tasks, warnings, true
}

func excelPreviewFromTasks(tasks []excelTask, warnings []string) ExcelImportPreview {
	preview := ExcelImportPreview{EventCount: len(tasks), Samples: []ExcelImportSample{}, Warnings: stringSlice(warnings)}
	seenRows := map[string]bool{}
	for index, task := range tasks {
		if index == 0 || task.Date.Before(preview.RangeStart) {
			preview.RangeStart = task.Date
		}
		if index == 0 || task.Date.After(preview.RangeEnd) {
			preview.RangeEnd = task.Date
		}
		seenRows[fmt.Sprintf("%s:%d", task.Sheet, task.Row)] = true
		if len(preview.Samples) < 5 {
			preview.Samples = append(preview.Samples, ExcelImportSample{Completed: excelTaskCompleted(task), Date: task.Date, Employee: task.Employee, Location: task.Location, Pop: task.Pop, Sheet: task.Sheet, Status: task.Status, Title: excelTaskTitle(task), Week: task.Week, Weekday: task.Weekday})
		}
	}
	preview.Rows = len(seenRows)
	return preview
}

func excelTaskTimeRange(date time.Time, options ExcelImportOptions) (time.Time, time.Time) {
	location, err := time.LoadLocation(normalizeTimezone("", options.Timezone))
	if err != nil {
		location = time.UTC
	}
	if options.AllDay {
		start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, location)
		return start, start.AddDate(0, 0, 1)
	}
	startClock := parseClock(options.WorkStart, "08:00")
	endClock := parseClock(options.WorkEnd, "17:00")
	start := time.Date(date.Year(), date.Month(), date.Day(), startClock.Hour(), startClock.Minute(), 0, 0, location)
	end := time.Date(date.Year(), date.Month(), date.Day(), endClock.Hour(), endClock.Minute(), 0, 0, location)
	if !end.After(start) {
		end = start.Add(time.Hour)
	}
	return start, end
}

func excelTaskTitle(task excelTask) string {
	if task.Summary != "" {
		return task.Summary
	}
	parts := []string{}
	if task.Pop != "" {
		parts = append(parts, task.Pop)
	}
	if task.Location != "" {
		parts = append(parts, task.Location)
	}
	if len(parts) == 0 {
		return "Excel-Task"
	}
	return strings.Join(parts, " · ")
}

func excelTaskDescription(task excelTask) string {
	lines := []string{
		fmt.Sprintf("Mitarbeiter: %s", valueOrDash(task.Employee)),
		fmt.Sprintf("KW: %d", task.Week),
		fmt.Sprintf("Wochentag: %s", task.Weekday),
		fmt.Sprintf("Quelle: %s, Zeile %d", task.Sheet, task.Row),
	}
	if task.Summary != "" {
		lines = append([]string{
			fmt.Sprintf("Bezeichnung / Kunde: %s", task.Summary),
			fmt.Sprintf("Projekt Nr.: %s", valueOrDash(task.Pop)),
		}, lines...)
	}
	if task.Status != "" {
		lines = append(lines, fmt.Sprintf("Status: %s", task.Status))
	}
	return strings.Join(lines, "\n")
}

func calendarImportReminders(startsAt time.Time, timezone string, calendar domain.Calendar) []domain.Reminder {
	if !calendar.ReminderEnabled {
		return nil
	}
	return remindersFromMinutes(calendarReminderMinutes(startsAt, timezone, calendar))
}

func excelEventsEqual(existing, imported domain.Event) bool {
	return existing.Title == imported.Title &&
		existing.Description == imported.Description &&
		existing.Location == imported.Location &&
		existing.StartsAt.Equal(imported.StartsAt) &&
		existing.EndsAt.Equal(imported.EndsAt) &&
		existing.Timezone == imported.Timezone &&
		existing.AllDay == imported.AllDay &&
		existing.Private == imported.Private &&
		existing.Completed == imported.Completed &&
		existing.Status == imported.Status
}

func filterImportableExcelTasks(tasks []excelTask) []excelTask {
	filtered := make([]excelTask, 0, len(tasks))
	for _, task := range tasks {
		if !excelTaskCancelled(task) {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func countCancelledExcelRows(tasks []excelTask) int {
	seenRows := map[string]bool{}
	for _, task := range tasks {
		if excelTaskCancelled(task) {
			seenRows[fmt.Sprintf("%s:%d", task.Sheet, task.Row)] = true
		}
	}
	return len(seenRows)
}

func excelTaskCancelled(task excelTask) bool {
	return strings.EqualFold(strings.TrimSpace(task.Status), "cancelled")
}

func excelTaskCompleted(task excelTask) bool {
	status := strings.ToLower(strings.TrimSpace(task.Status))
	return status != "open" && status != "cancelled"
}

func excelTaskUID(task excelTask) string {
	raw := fmt.Sprintf("%s-%d-%d-%s-%s-%s-%s", task.Sheet, task.Row, task.Week, task.Weekday, task.Pop, task.Location, task.Summary)
	raw = strings.ToLower(strings.TrimSpace(raw))
	replacer := strings.NewReplacer(" ", "-", "·", "-", "/", "-", "\\", "-", ":", "-", ";", "-", "@", "-")
	return "excel-" + replacer.Replace(raw) + "@calendaradvanced"
}

func yearFromSheetName(name string) (int, bool) {
	match := sheetYearPattern.FindString(name)
	if match == "" {
		return 0, false
	}
	year, err := strconv.Atoi(match)
	return year, err == nil
}

func weekFromCell(value string) (int, bool) {
	value = strings.TrimSpace(strings.TrimSuffix(value, ".0"))
	if value == "" {
		return 0, false
	}
	week, err := strconv.Atoi(value)
	return week, err == nil && week >= 1 && week <= 53
}

func isoWeekdayDate(year, week, dayOffset int, location *time.Location) (time.Time, error) {
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, location)
	isoWeekday := int(jan4.Weekday())
	if isoWeekday == 0 {
		isoWeekday = 7
	}
	weekOneMonday := jan4.AddDate(0, 0, 1-isoWeekday)
	date := weekOneMonday.AddDate(0, 0, (week-1)*7+dayOffset)
	if actualYear, actualWeek := date.ISOWeek(); actualYear != year || actualWeek != week {
		return time.Time{}, fmt.Errorf("invalid iso week")
	}
	return date, nil
}

func isMarked(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "x"
}

func weekdayName(offset int) string {
	return []string{"Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag", "Sonntag"}[offset]
}

func looksLikeMontagePlanningSheet(sheet xlsx.Sheet) bool {
	for _, row := range sheet.Rows {
		if row.Index != 2 {
			continue
		}
		return strings.EqualFold(cell(row.Cells, 0), "Mitarbeiter") &&
			strings.EqualFold(cell(row.Cells, 1), "Projekt Nr.") &&
			strings.Contains(strings.ToLower(cell(row.Cells, 2)), "bezeichnung")
	}
	return false
}

func montagePlanningWeek(sheet xlsx.Sheet, sourceName string) (int, bool) {
	if week, ok := weekFromFilename(sourceName); ok {
		return week, true
	}
	for _, row := range sheet.Rows {
		if row.Index == 1 && strings.EqualFold(cell(row.Cells, 0), "KW") {
			return weekFromCell(cell(row.Cells, 1))
		}
	}
	return 0, false
}

func weekFromFilename(sourceName string) (int, bool) {
	match := montageWeekPattern.FindStringSubmatch(sourceName)
	if len(match) != 2 {
		return 0, false
	}
	return weekFromCell(match[1])
}

func montagePlanningYear(week int, now time.Time) int {
	year, currentWeek := now.ISOWeek()
	if currentWeek <= 4 && week >= 50 {
		return year - 1
	}
	if currentWeek >= 50 && week <= 4 {
		return year + 1
	}
	return year
}

func cell(cells []string, index int) string {
	if index < 0 || index >= len(cells) {
		return ""
	}
	return strings.TrimSpace(cells[index])
}

func parseClock(value, fallback string) time.Time {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err == nil {
		return parsed
	}
	parsed, _ = time.Parse("15:04", fallback)
	return parsed
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
