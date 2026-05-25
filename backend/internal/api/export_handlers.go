package api

import (
	"fmt"
	"net/http"
	"time"

	"calendaradvanced/internal/application"
)

func (s *Server) exportCSV(w http.ResponseWriter, r *http.Request) {
	file, err := s.Services.ExcelExports.ExportCSV(CurrentUser(r), exportInputFromQuery(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeFile(w, file)
}

func (s *Server) exportXLSX(w http.ResponseWriter, r *http.Request) {
	file, err := s.Services.ExcelExports.ExportXLSX(CurrentUser(r), exportInputFromQuery(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeFile(w, file)
}

func exportInputFromQuery(r *http.Request) application.ExportInput {
	query := r.URL.Query()
	from, _ := time.Parse(time.RFC3339, query.Get("from"))
	to, _ := time.Parse(time.RFC3339, query.Get("to"))
	kind := query.Get("kind")
	if kind == "" {
		kind = "events"
	}
	return application.ExportInput{Kind: kind, From: from, To: to, Query: query.Get("q")}
}

func writeFile(w http.ResponseWriter, file application.ExportFile) {
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.FileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Data)
}
