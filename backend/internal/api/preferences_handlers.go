package api

import (
	"net/http"

	"calendaradvanced/internal/application"
	"calendaradvanced/internal/domain"
)

func (s *Server) getPreferences(w http.ResponseWriter, r *http.Request) {
	result, err := s.Services.Preferences.Get(CurrentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) savePreferences(w http.ResponseWriter, r *http.Request) {
	var input domain.GeneralPreferences
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	result, err := s.Services.Preferences.Save(CurrentUser(r), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
