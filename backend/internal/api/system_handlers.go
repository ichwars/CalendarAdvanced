package api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
)

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "app": s.Services.Config.AppName, "version": s.Services.Config.Version})
}

func (s *Server) updateCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Services.Updates.Check())
}

func (s *Server) auditLog(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	entries, err := s.Services.Audit.List(limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": entries})
}

func (s *Server) exportBackup(w http.ResponseWriter, r *http.Request) {
	payload, err := s.Services.Backup.Export(CurrentUser(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, "calendaradvanced-backup.json"))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func (s *Server) previewBackupRestore(w http.ResponseWriter, r *http.Request) {
	payload, err := readBackupUpload(r)
	if err != nil {
		writeError(w, err)
		return
	}
	preview, err := s.Services.Backup.PreviewRestore(payload)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) restoreBackup(w http.ResponseWriter, r *http.Request) {
	payload, err := readBackupUpload(r)
	if err != nil {
		writeError(w, err)
		return
	}
	preview, err := s.Services.Backup.Restore(CurrentUser(r), payload, clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func readBackupUpload(r *http.Request) ([]byte, error) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		return nil, err
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, 20<<20))
}
