package application

import (
	"calendaradvanced/internal/infrastructure/sqlite"
)

type SharingService struct {
	Store *sqlite.Store
	Audit *AuditService
}
