package api

import (
	"net/http"

	"calendaradvanced/internal/application"
	"calendaradvanced/internal/domain"
)

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Services.Users.List()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": users})
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var input application.CreateUserInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	user, err := s.Services.Users.Create(input, CurrentUser(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) updateUserRoles(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Roles []domain.RoleName `json:"roles"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	if err := s.Services.Users.SetRoles(parseID(r.PathValue("id")), input.Roles, CurrentUser(r), clientIP(r), userAgent(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
