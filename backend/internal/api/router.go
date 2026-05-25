package api

import (
	"errors"
	"net/http"

	"calendaradvanced/internal/application"
	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/filesystem"
)

const (
	sessionCookieName = "ck_session"
	csrfCookieName    = "ck_csrf"
)

var (
	ErrAuth                   = application.ErrUnauthorized
	ErrForbidden              = application.ErrForbidden
	ErrUnsupportedContentType = errors.New("unsupported_content_type")
)

type Server struct {
	Services *application.Services
}

func NewServer(services *application.Services) http.Handler {
	server := &Server{Services: services}
	mux := http.NewServeMux()
	server.routes(mux)
	var handler http.Handler = mux
	handler = server.securityHeaders(handler)
	return handler
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/v1/setup/status", s.setupStatus)
	mux.HandleFunc("POST /api/v1/setup/admin", s.setupAdmin)

	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("POST /api/v1/auth/logout", s.requireWrite(s.logout))
	mux.HandleFunc("GET /api/v1/auth/me", s.requireAuth(s.me))
	mux.HandleFunc("POST /api/v1/auth/password/change", s.requireWrite(s.changePassword))
	mux.HandleFunc("POST /api/v1/auth/password/request-reset", s.requestPasswordReset)
	mux.HandleFunc("POST /api/v1/auth/password/reset", s.resetPassword)

	mux.HandleFunc("POST /api/v1/security/2fa/setup", s.requireWrite(s.twoFactorSetup))
	mux.HandleFunc("POST /api/v1/security/2fa/enable", s.requireWrite(s.twoFactorEnable))
	mux.HandleFunc("POST /api/v1/security/2fa/disable", s.requireWrite(s.twoFactorDisable))

	mux.HandleFunc("GET /api/v1/users", s.requireRole(domain.RoleAdmin, s.listUsers))
	mux.HandleFunc("POST /api/v1/users", s.requireRoleWrite(domain.RoleAdmin, s.createUser))
	mux.HandleFunc("PUT /api/v1/users/{id}/roles", s.requireRoleWrite(domain.RoleAdmin, s.updateUserRoles))

	mux.HandleFunc("GET /api/v1/preferences", s.requireAuth(s.getPreferences))
	mux.HandleFunc("PUT /api/v1/preferences", s.requireWrite(s.savePreferences))

	mux.HandleFunc("GET /api/v1/calendars", s.requireAuth(s.listCalendars))
	mux.HandleFunc("POST /api/v1/calendars", s.requireRoleWrite(domain.RoleEditor, s.createCalendar))
	mux.HandleFunc("PUT /api/v1/calendars/{id}", s.requireRoleWrite(domain.RoleEditor, s.updateCalendar))
	mux.HandleFunc("DELETE /api/v1/calendars/{id}", s.requireRoleWrite(domain.RoleEditor, s.deleteCalendar))

	mux.HandleFunc("GET /api/v1/events", s.requireAuth(s.listEvents))
	mux.HandleFunc("POST /api/v1/events", s.requireRoleWrite(domain.RoleEditor, s.createEvent))
	mux.HandleFunc("PUT /api/v1/events/{id}", s.requireRoleWrite(domain.RoleEditor, s.updateEvent))
	mux.HandleFunc("DELETE /api/v1/events/{id}", s.requireRoleWrite(domain.RoleEditor, s.deleteEvent))
	mux.HandleFunc("GET /api/v1/tasks", s.requireAuth(s.listTasks))
	mux.HandleFunc("POST /api/v1/tasks", s.requireWrite(s.createTask))
	mux.HandleFunc("PUT /api/v1/tasks/{id}", s.requireWrite(s.updateTask))
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", s.requireWrite(s.deleteTask))
	mux.HandleFunc("POST /api/v1/tasks/{id}/reminder-delivered", s.requireWrite(s.markTaskReminderDelivered))
	mux.HandleFunc("GET /api/v1/contacts", s.requireAuth(s.listContacts))
	mux.HandleFunc("POST /api/v1/contacts", s.requireWrite(s.createContact))
	mux.HandleFunc("PUT /api/v1/contacts/{id}", s.requireWrite(s.updateContact))
	mux.HandleFunc("DELETE /api/v1/contacts/{id}", s.requireWrite(s.deleteContact))
	mux.HandleFunc("GET /api/v1/reminders/due", s.requireAuth(s.dueReminders))
	mux.HandleFunc("POST /api/v1/reminders/{id}/delivered", s.requireWrite(s.markReminderDelivered))

	mux.HandleFunc("GET /api/v1/integrations/caldav/tokens", s.requireAuth(s.listCalDAVTokens))
	mux.HandleFunc("POST /api/v1/integrations/caldav/tokens", s.requireWrite(s.createCalDAVToken))
	mux.HandleFunc("GET /api/v1/integrations/caldav/connection", s.requireAuth(s.getCalDAVConnection))
	mux.HandleFunc("PUT /api/v1/integrations/caldav/connection", s.requireWrite(s.saveCalDAVConnection))
	mux.HandleFunc("POST /api/v1/integrations/caldav/test", s.requireWrite(s.testCalDAVConnection))
	mux.HandleFunc("GET /api/v1/integrations/caldav/collections", s.requireAuth(s.listDAVCollections))
	mux.HandleFunc("GET /api/v1/integrations/caldav/sync-history", s.requireAuth(s.listDAVSyncHistory))
	mux.HandleFunc("POST /api/v1/integrations/caldav/collections/discover", s.requireWrite(s.discoverDAVCollections))
	mux.HandleFunc("PUT /api/v1/integrations/caldav/collections", s.requireWrite(s.saveDAVCollectionSelections))
	mux.HandleFunc("POST /api/v1/integrations/caldav/sync", s.requireWrite(s.syncDAVNow))

	mux.HandleFunc("GET /api/v1/exports/csv", s.requireAuth(s.exportCSV))
	mux.HandleFunc("GET /api/v1/exports/xlsx", s.requireAuth(s.exportXLSX))
	mux.HandleFunc("POST /api/v1/imports/excel/preview", s.requireRoleWrite(domain.RoleEditor, s.previewExcelImport))
	mux.HandleFunc("POST /api/v1/imports/excel", s.requireRoleWrite(domain.RoleEditor, s.importExcel))
	mux.HandleFunc("POST /api/v1/imports/ics/preview", s.requireRoleWrite(domain.RoleEditor, s.previewICSImport))
	mux.HandleFunc("POST /api/v1/imports/ics", s.requireRoleWrite(domain.RoleEditor, s.importICS))

	mux.HandleFunc("GET /api/v1/audit", s.requireRole(domain.RoleAdmin, s.auditLog))
	mux.HandleFunc("GET /api/v1/system/update-check", s.requireRole(domain.RoleAdmin, s.updateCheck))
	mux.HandleFunc("GET /api/v1/system/backup", s.requireRole(domain.RoleAdmin, s.exportBackup))
	mux.HandleFunc("POST /api/v1/system/backup/preview-restore", s.requireRoleWrite(domain.RoleAdmin, s.previewBackupRestore))
	mux.HandleFunc("POST /api/v1/system/backup/restore", s.requireRoleWrite(domain.RoleAdmin, s.restoreBackup))

	mux.HandleFunc("GET /.well-known/caldav", s.caldavWellKnown)
	mux.HandleFunc("PROPFIND /dav/", s.caldav)
	mux.HandleFunc("REPORT /dav/", s.caldav)
	mux.HandleFunc("GET /dav/", s.caldav)
	mux.Handle("/", filesystem.SPAHandler{Dir: s.Services.Config.StaticDir})
}
