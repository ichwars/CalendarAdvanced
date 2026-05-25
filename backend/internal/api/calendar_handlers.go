package api

import (
	"net/http"

	"calendaradvanced/internal/application"
)

func (s *Server) listCalendars(w http.ResponseWriter, r *http.Request) {
	items, err := s.Services.Calendars.List(CurrentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createCalendar(w http.ResponseWriter, r *http.Request) {
	var input application.CalendarInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	calendar, err := s.Services.Calendars.Create(input, CurrentUser(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, calendar)
}

func (s *Server) updateCalendar(w http.ResponseWriter, r *http.Request) {
	var input application.CalendarInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	calendar, err := s.Services.Calendars.Update(parseID(r.PathValue("id")), input, CurrentUser(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, calendar)
}

func (s *Server) deleteCalendar(w http.ResponseWriter, r *http.Request) {
	if err := s.Services.Calendars.Delete(parseID(r.PathValue("id")), CurrentUser(r), clientIP(r), userAgent(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
