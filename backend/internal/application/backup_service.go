package application

import (
	"encoding/json"
	"time"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type BackupService struct {
	Store *sqlite.Store
	Audit *AuditService
}

type BackupEnvelope struct {
	App       string         `json:"app"`
	Version   int            `json:"version"`
	CreatedAt time.Time      `json:"createdAt"`
	Data      map[string]any `json:"data"`
}

type BackupPreview struct {
	App       string         `json:"app"`
	Version   int            `json:"version"`
	CreatedAt time.Time      `json:"createdAt"`
	Counts    map[string]int `json:"counts"`
}

func (s *BackupService) Export(user domain.User, ip, userAgent string) ([]byte, error) {
	data, err := s.Store.ListAppDataForBackup()
	if err != nil {
		return nil, err
	}
	envelope := BackupEnvelope{App: "CalendarAdvanced", Version: 1, CreatedAt: time.Now().UTC(), Data: data}
	encoded, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, err
	}
	s.Audit.Record(user.ID, domain.AuditBackupCreated, "backup", "json", ip, userAgent, nil)
	return encoded, nil
}

func (s *BackupService) PreviewRestore(payload []byte) (BackupPreview, error) {
	envelope, err := parseBackupEnvelope(payload)
	if err != nil {
		return BackupPreview{}, err
	}
	return BackupPreview{App: envelope.App, Version: envelope.Version, CreatedAt: envelope.CreatedAt, Counts: backupCounts(envelope.Data)}, nil
}

func (s *BackupService) ValidateRestore(payload []byte) error {
	_, err := parseBackupEnvelope(payload)
	return err
}

func (s *BackupService) Restore(user domain.User, payload []byte, ip, userAgent string) (BackupPreview, error) {
	envelope, err := parseBackupEnvelope(payload)
	if err != nil {
		return BackupPreview{}, err
	}
	if err := s.Store.RestoreAppDataFromBackup(envelope.Data); err != nil {
		return BackupPreview{}, err
	}
	preview := BackupPreview{App: envelope.App, Version: envelope.Version, CreatedAt: envelope.CreatedAt, Counts: backupCounts(envelope.Data)}
	s.Audit.Record(user.ID, domain.AuditBackupRestored, "backup", "json", ip, userAgent, preview.Counts)
	return preview, nil
}

func parseBackupEnvelope(payload []byte) (BackupEnvelope, error) {
	var envelope BackupEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return BackupEnvelope{}, err
	}
	if envelope.App != "CalendarAdvanced" || envelope.Version < 1 || envelope.Version > 2 {
		return BackupEnvelope{}, NewError("backup_version_unsupported", "Backup-Version wird nicht unterstützt.", nil)
	}
	if envelope.Data == nil {
		return BackupEnvelope{}, NewError("backup_empty", "Backup enthält keine wiederherstellbaren Daten.", nil)
	}
	return envelope, nil
}

func backupCounts(data map[string]any) map[string]int {
	counts := map[string]int{}
	for key, value := range data {
		if rows, ok := value.([]any); ok {
			counts[key] = len(rows)
		}
	}
	return counts
}
