package api

import (
	"net/http"

	"calendaradvanced/internal/application"
)

func (s *Server) listCalDAVTokens(w http.ResponseWriter, r *http.Request) {
	items, err := s.Services.CalDAV.ListTokens(CurrentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createCalDAVToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	result, err := s.Services.CalDAV.CreateToken(CurrentUser(r), input.Name, clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) getCalDAVConnection(w http.ResponseWriter, r *http.Request) {
	connection, err := s.Services.CalDAV.GetConnection(CurrentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, connection)
}

func (s *Server) saveCalDAVConnection(w http.ResponseWriter, r *http.Request) {
	var input application.CalDAVConnectionInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	connection, err := s.Services.CalDAV.SaveConnection(CurrentUser(r), input, clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, connection)
}

func (s *Server) testCalDAVConnection(w http.ResponseWriter, r *http.Request) {
	var input application.CalDAVConnectionInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	result, err := s.Services.CalDAV.TestConnection(CurrentUser(r), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listDAVCollections(w http.ResponseWriter, r *http.Request) {
	items, err := s.Services.CalDAV.ListCollections(CurrentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) listDAVSyncHistory(w http.ResponseWriter, r *http.Request) {
	items, err := s.Services.CalDAV.ListSyncHistory(CurrentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) discoverDAVCollections(w http.ResponseWriter, r *http.Request) {
	var input application.CalDAVConnectionInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	result, err := s.Services.CalDAV.DiscoverCollections(CurrentUser(r), input, clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) saveDAVCollectionSelections(w http.ResponseWriter, r *http.Request) {
	var input application.DAVCollectionSelectionInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	items, err := s.Services.CalDAV.SaveCollectionSelections(CurrentUser(r), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) syncDAVNow(w http.ResponseWriter, r *http.Request) {
	var input application.DAVSyncInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	result, err := s.Services.CalDAV.SyncNow(CurrentUser(r), clientIP(r), userAgent(r), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
