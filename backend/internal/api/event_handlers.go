package api

import (
	"net/http"
	"strconv"
	"time"

	"calendaradvanced/internal/application"
)

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	from, _ := time.Parse(time.RFC3339, query.Get("from"))
	to, _ := time.Parse(time.RFC3339, query.Get("to"))
	calendarID, _ := strconv.ParseInt(query.Get("calendarId"), 10, 64)
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	expand := query.Get("expand") == "true"
	events, err := s.Services.Events.List(CurrentUser(r), application.EventListInput{From: from, To: to, CalendarID: calendarID, Query: query.Get("q"), Limit: limit, Offset: offset, Expand: expand})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": events})
}

func (s *Server) createEvent(w http.ResponseWriter, r *http.Request) {
	var input application.EventInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	event, err := s.Services.Events.Create(input, CurrentUser(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

func (s *Server) updateEvent(w http.ResponseWriter, r *http.Request) {
	var input application.EventInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	event, err := s.Services.Events.Update(parseID(r.PathValue("id")), input, CurrentUser(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (s *Server) deleteEvent(w http.ResponseWriter, r *http.Request) {
	if err := s.Services.Events.Delete(parseID(r.PathValue("id")), CurrentUser(r), clientIP(r), userAgent(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
