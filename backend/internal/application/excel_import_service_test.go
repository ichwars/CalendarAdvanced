package application

import (
	"strings"
	"testing"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/excel"
	"calendaradvanced/internal/infrastructure/sqlite"
)

func TestParseExcelTasksUsesSheetYearWeekColumnAndWeekdayMarkers(t *testing.T) {
	row := make([]string, 22)
	row[0] = "22"
	row[1] = "fra-901"
	row[3] = "Frankfurt am Main"
	row[15] = "Daniel R."
	row[16] = "x"
	row[17] = "x"
	row[21] = "open"
	data, err := excel.BuildSimpleXLSX("2026 week 22-29", [][]string{row})
	if err != nil {
		t.Fatal(err)
	}

	tasks, warnings, err := parseExcelTasks(data, "Europe/Berlin", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Date.Format("2006-01-02") != "2026-05-25" || tasks[0].Weekday != "Montag" {
		t.Fatalf("unexpected first task date/weekday: %s %s", tasks[0].Date.Format("2006-01-02"), tasks[0].Weekday)
	}
	if tasks[1].Date.Format("2006-01-02") != "2026-05-26" || tasks[1].Weekday != "Dienstag" {
		t.Fatalf("unexpected second task date/weekday: %s %s", tasks[1].Date.Format("2006-01-02"), tasks[1].Weekday)
	}
	if tasks[0].Pop != "fra-901" || tasks[0].Location != "Frankfurt am Main" || tasks[0].Employee != "Daniel R." || tasks[0].Status != "open" {
		t.Fatalf("unexpected task fields: %+v", tasks[0])
	}
}

func TestExcelImportSkipsUnchangedAndUpdatesChangedRows(t *testing.T) {
	services := testServices(t)
	user := createTestUser(t, services)
	calendar, err := services.Store.CreateCalendar(domain.Calendar{OwnerUserID: user.ID, Name: "Import", Color: "#ffcd00", Timezone: "Europe/Berlin"})
	if err != nil {
		t.Fatal(err)
	}
	options := ExcelImportOptions{CalendarID: calendar.ID, Timezone: "Europe/Berlin", WorkStart: "08:00", WorkEnd: "17:00"}
	data := excelImportFixture(t, "open")

	first, err := services.ExcelImports.Import(data, options, user, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if first.Imported != 1 || first.Updated != 0 || first.Skipped != 0 {
		t.Fatalf("first import result = %+v", first)
	}

	second, err := services.ExcelImports.Import(data, options, user, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if second.Imported != 0 || second.Updated != 0 || second.Skipped != 1 {
		t.Fatalf("second import result = %+v", second)
	}

	changed, err := services.ExcelImports.Import(excelImportFixture(t, "closed"), options, user, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if changed.Imported != 0 || changed.Updated != 1 || changed.Skipped != 0 {
		t.Fatalf("changed import result = %+v", changed)
	}
	events, err := services.Store.ListEvents(sqlite.EventFilter{UserID: user.ID, CalendarID: calendar.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one deduplicated event, got %d", len(events))
	}
	if events[0].Description == "" || !strings.Contains(events[0].Description, "Status: closed") {
		t.Fatalf("event was not updated: %q", events[0].Description)
	}
	if !events[0].Completed {
		t.Fatalf("expected closed Excel status to mark event completed")
	}
}

func TestExcelImportSkipsCancelledStatus(t *testing.T) {
	services := testServices(t)
	user := createTestUser(t, services)
	calendar, err := services.Store.CreateCalendar(domain.Calendar{OwnerUserID: user.ID, Name: "Import", Color: "#ffcd00", Timezone: "Europe/Berlin"})
	if err != nil {
		t.Fatal(err)
	}
	options := ExcelImportOptions{CalendarID: calendar.ID, Timezone: "Europe/Berlin", WorkStart: "08:00", WorkEnd: "17:00"}
	if _, err := services.ExcelImports.Import(excelImportFixture(t, "open"), options, user, "127.0.0.1", "test"); err != nil {
		t.Fatal(err)
	}

	result, err := services.ExcelImports.Import(excelImportFixture(t, "cancelled"), options, user, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 0 || result.Updated != 0 || result.Skipped != 1 || result.SkippedCancelled != 1 {
		t.Fatalf("cancelled import result = %+v", result)
	}
	events, err := services.Store.ListEvents(sqlite.EventFilter{UserID: user.ID, CalendarID: calendar.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected cancelled row to remove prior import, got %d events", len(events))
	}
}

func TestExcelImportEmployeeFilterUsesColumnP(t *testing.T) {
	rowA := make([]string, 22)
	rowA[0] = "22"
	rowA[1] = "fra-901"
	rowA[3] = "Frankfurt am Main"
	rowA[15] = "Daniel R."
	rowA[16] = "x"
	rowB := make([]string, 22)
	rowB[0] = "22"
	rowB[1] = "ber-101"
	rowB[3] = "Berlin"
	rowB[15] = "Mara K."
	rowB[16] = "x"
	data, err := excel.BuildSimpleXLSX("2026 week 22-29", [][]string{rowA, rowB})
	if err != nil {
		t.Fatal(err)
	}

	service := &ExcelImportService{}
	preview, err := service.Preview(data, "Europe/Berlin", "mara", "")
	if err != nil {
		t.Fatal(err)
	}
	if preview.EventCount != 1 || len(preview.Samples) != 1 {
		t.Fatalf("preview = %+v", preview)
	}
	if preview.Samples[0].Employee != "Mara K." || preview.Samples[0].Pop != "ber-101" {
		t.Fatalf("unexpected filtered sample: %+v", preview.Samples[0])
	}
}

func TestMontagePlanningImportUsesProjAllgEmployeeColumnAndSeparateProjects(t *testing.T) {
	rows := [][]string{
		{"KW", "27"},
		{"Mitarbeiter", "Projekt Nr.", "Bezeichnung / Kunde", "Mo", "Di", "Mi", "Do", "Fr", "Sa", "So"},
		{"Rothe, Daniel", "26285013", "WLAN Verkabelung ENB Haus I", "x", "x", "", "", "", "", ""},
		{"Rothe, Daniel", "25224019", "Städt. Klinikum DE DECT-System", "x", "x", "", "", "", "", ""},
		{"Andere, Person", "999", "Nicht importieren", "x", "", "", "", "", "", ""},
	}
	data, err := excel.BuildSimpleXLSX("PROJALLG", rows)
	if err != nil {
		t.Fatal(err)
	}

	service := &ExcelImportService{}
	preview, err := service.Preview(data, "Europe/Berlin", "Rothe, Daniel", "Montageplanung KW27.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if preview.EventCount != 4 || preview.Rows != 2 {
		t.Fatalf("preview = %+v", preview)
	}
	if preview.Samples[0].Title != "WLAN Verkabelung ENB Haus I" || preview.Samples[0].Pop != "26285013" {
		t.Fatalf("unexpected first sample: %+v", preview.Samples[0])
	}
	if preview.Samples[1].Title != "WLAN Verkabelung ENB Haus I" || preview.Samples[2].Title != "Städt. Klinikum DE DECT-System" {
		t.Fatalf("expected separate appointments per project/day, got %+v", preview.Samples)
	}
}

func excelImportFixture(t *testing.T, status string) []byte {
	t.Helper()
	row := make([]string, 22)
	row[0] = "22"
	row[1] = "fra-901"
	row[3] = "Frankfurt am Main"
	row[15] = "Daniel R."
	row[16] = "x"
	row[21] = status
	data, err := excel.BuildSimpleXLSX("2026 week 22-29", [][]string{row})
	if err != nil {
		t.Fatal(err)
	}
	return data
}
