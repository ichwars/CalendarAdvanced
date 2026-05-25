package application

import (
	"calendaradvanced/internal/infrastructure/config"
	"calendaradvanced/internal/infrastructure/crypto"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type Services struct {
	Config       config.Config
	Store        *sqlite.Store
	Cipher       *crypto.TokenCipher
	RateLimit    *RateLimitService
	Audit        *AuditService
	Setup        *SetupService
	Auth         *AuthService
	Users        *UserService
	Preferences  *PreferencesService
	Calendars    *CalendarService
	Events       *EventService
	Tasks        *TaskService
	Contacts     *ContactService
	Recurrence   *RecurrenceService
	Reminders    *ReminderService
	Sharing      *SharingService
	CalDAV       *CalDAVService
	ExcelImports *ExcelImportService
	ExcelExports *ExcelExportService
	ICSImports   *ICSImportService
	Backup       *BackupService
	Updates      *UpdateService
}

func NewServices(cfg config.Config, store *sqlite.Store) *Services {
	cipher, _ := crypto.NewTokenCipher(cfg.TokenEncryptionKey)
	rateLimiter := NewRateLimitService()
	audit := &AuditService{Store: store}
	services := &Services{Config: cfg, Store: store, Cipher: cipher, RateLimit: rateLimiter, Audit: audit}
	services.Setup = &SetupService{Store: store, Audit: audit}
	services.Auth = &AuthService{Config: cfg, Store: store, Audit: audit, RateLimit: rateLimiter, Cipher: cipher}
	services.Users = &UserService{Store: store, Audit: audit}
	services.Preferences = &PreferencesService{Store: store}
	services.Calendars = &CalendarService{Store: store, Audit: audit}
	services.Recurrence = &RecurrenceService{}
	services.Events = &EventService{Store: store, Audit: audit, Recurrence: services.Recurrence}
	services.Tasks = &TaskService{Store: store, Audit: audit}
	services.Contacts = &ContactService{Store: store, Audit: audit}
	services.Reminders = &ReminderService{Store: store}
	services.Sharing = &SharingService{Store: store, Audit: audit}
	services.CalDAV = &CalDAVService{Store: store, Audit: audit, Cipher: cipher}
	services.ExcelImports = &ExcelImportService{Store: store, Audit: audit}
	services.ExcelExports = &ExcelExportService{Store: store, Audit: audit}
	services.ICSImports = &ICSImportService{Store: store, Audit: audit}
	services.Backup = &BackupService{Store: store, Audit: audit}
	services.Updates = &UpdateService{Config: cfg}
	return services
}
