package api

import (
	"net/http"
	"strconv"

	"calendaradvanced/internal/application"
)

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	var completed *bool
	if query.Get("completed") != "" {
		value := query.Get("completed") == "true"
		completed = &value
	}
	tasks, err := s.Services.Tasks.List(CurrentUser(r), application.TaskListInput{Query: query.Get("q"), Completed: completed, Limit: limit, Offset: offset})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tasks})
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var input application.TaskInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	task, err := s.Services.Tasks.Create(input, CurrentUser(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	var input application.TaskInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	task, err := s.Services.Tasks.Update(parseID(r.PathValue("id")), input, CurrentUser(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	if err := s.Services.Tasks.Delete(parseID(r.PathValue("id")), CurrentUser(r), clientIP(r), userAgent(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) markTaskReminderDelivered(w http.ResponseWriter, r *http.Request) {
	if err := s.Services.Tasks.MarkReminderDelivered(parseID(r.PathValue("id")), CurrentUser(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
