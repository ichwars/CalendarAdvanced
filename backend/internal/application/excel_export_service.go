package application

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/excel"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type ExcelExportService struct {
	Store *sqlite.Store
	Audit *AuditService
}

type ExportInput struct {
	Kind  string
	From  time.Time
	To    time.Time
	Query string
}

type ExportFile struct {
	FileName    string
	ContentType string
	Data        []byte
}

func (s *ExcelExportService) ExportCSV(user domain.User, input ExportInput, ip, userAgent string) (ExportFile, error) {
	events, err := s.Store.ListEvents(sqlite.EventFilter{UserID: user.ID, From: input.From, To: input.To, Query: input.Query, Limit: 1000})
	if err != nil {
		return ExportFile{}, err
	}
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	_ = writer.Write([]string{"Title", "Start", "End", "Timezone", "Location", "Attendees"})
	for _, event := range events {
		_ = writer.Write([]string{event.Title, event.StartsAt.Format(time.RFC3339), event.EndsAt.Format(time.RFC3339), event.Timezone, event.Location, fmt.Sprint(len(event.Attendees))})
	}
	writer.Flush()
	_ = s.Store.RecordExcelExport(domain.ExcelExport{UserID: user.ID, Kind: input.Kind, Format: "csv", RangeStart: input.From, RangeEnd: input.To})
	s.Audit.Record(user.ID, domain.AuditExportCreated, "export", "csv", ip, userAgent, map[string]any{"kind": input.Kind})
	return ExportFile{FileName: "calendaradvanced-events.csv", ContentType: "text/csv; charset=utf-8", Data: buf.Bytes()}, nil
}

func (s *ExcelExportService) ExportXLSX(user domain.User, input ExportInput, ip, userAgent string) (ExportFile, error) {
	events, err := s.Store.ListEvents(sqlite.EventFilter{UserID: user.ID, From: input.From, To: input.To, Query: input.Query, Limit: 1000})
	if err != nil {
		return ExportFile{}, err
	}
	rows := [][]string{{"Title", "Start", "End", "Timezone", "Location", "Attendees"}}
	for _, event := range events {
		rows = append(rows, []string{event.Title, event.StartsAt.Format(time.RFC3339), event.EndsAt.Format(time.RFC3339), event.Timezone, event.Location, fmt.Sprint(len(event.Attendees))})
	}
	data, err := excel.BuildSimpleXLSX("Events", rows)
	if err != nil {
		return ExportFile{}, err
	}
	_ = s.Store.RecordExcelExport(domain.ExcelExport{UserID: user.ID, Kind: input.Kind, Format: "xlsx", RangeStart: input.From, RangeEnd: input.To})
	s.Audit.Record(user.ID, domain.AuditExportCreated, "export", "xlsx", ip, userAgent, map[string]any{"kind": input.Kind})
	return ExportFile{FileName: "calendaradvanced-events.xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Data: data}, nil
}
