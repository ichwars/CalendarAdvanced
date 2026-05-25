package api

import (
	"net/http"
	"time"
)

func (s *Server) dueReminders(w http.ResponseWriter, r *http.Request) {
	reminders, err := s.Services.Reminders.Due(CurrentUser(r), time.Now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": reminders})
}

func (s *Server) markReminderDelivered(w http.ResponseWriter, r *http.Request) {
	if err := s.Services.Reminders.MarkDelivered(CurrentUser(r), parseID(r.PathValue("id"))); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
