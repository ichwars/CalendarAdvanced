package api

import (
	"net/http"
	"strconv"

	"calendaradvanced/internal/application"
)

func (s *Server) listContacts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	contacts, err := s.Services.Contacts.List(CurrentUser(r), application.ContactListInput{Query: query.Get("q"), Limit: limit, Offset: offset})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": contacts})
}

func (s *Server) createContact(w http.ResponseWriter, r *http.Request) {
	var input application.ContactInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	contact, err := s.Services.Contacts.Create(input, CurrentUser(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, contact)
}

func (s *Server) updateContact(w http.ResponseWriter, r *http.Request) {
	var input application.ContactInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	contact, err := s.Services.Contacts.Update(parseID(r.PathValue("id")), input, CurrentUser(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contact)
}

func (s *Server) deleteContact(w http.ResponseWriter, r *http.Request) {
	if err := s.Services.Contacts.Delete(parseID(r.PathValue("id")), CurrentUser(r), clientIP(r), userAgent(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
