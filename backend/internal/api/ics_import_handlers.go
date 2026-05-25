package api

import (
	"io"
	"net/http"
	"strconv"

	"calendaradvanced/internal/application"
)

func (s *Server) previewICSImport(w http.ResponseWriter, r *http.Request) {
	data, timezone, ok := readICSUpload(w, r)
	if !ok {
		return
	}
	result, err := s.Services.ICSImports.Preview(data, timezone)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) importICS(w http.ResponseWriter, r *http.Request) {
	data, timezone, ok := readICSUpload(w, r)
	if !ok {
		return
	}
	calendarID, _ := strconv.ParseInt(r.FormValue("calendarId"), 10, 64)
	result, err := s.Services.ICSImports.Import(data, calendarID, CurrentUser(r), timezone, clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func readICSUpload(w http.ResponseWriter, r *http.Request) ([]byte, string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, application.MaxICSImportBytes()+1024)
	if err := r.ParseMultipartForm(application.MaxICSImportBytes()); err != nil {
		writeError(w, application.NewError("invalid_import", "ICS-Datei konnte nicht gelesen werden", nil))
		return nil, "", false
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, application.NewError("invalid_import", "ICS-Datei fehlt", nil))
		return nil, "", false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, application.MaxICSImportBytes()+1))
	if err != nil || int64(len(data)) > application.MaxICSImportBytes() {
		writeError(w, application.NewError("invalid_import", "ICS-Datei ist zu groß", nil))
		return nil, "", false
	}
	return data, r.FormValue("timezone"), true
}
