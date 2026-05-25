package application

import (
	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type AuditService struct {
	Store *sqlite.Store
}

func (s *AuditService) Record(actorID int64, action domain.AuditAction, entityType, entityID, ip, userAgent string, metadata any) {
	if s == nil || s.Store == nil {
		return
	}
	_ = s.Store.Audit(actorID, action, entityType, entityID, ip, userAgent, metadata)
}

func (s *AuditService) List(limit, offset int) ([]domain.AuditEntry, error) {
	return s.Store.ListAudit(limit, offset)
}
