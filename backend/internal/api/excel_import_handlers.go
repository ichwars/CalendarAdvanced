package api

import (
	"io"
	"net/http"
	"strconv"

	"calendaradvanced/internal/application"
)

func (s *Server) previewExcelImport(w http.ResponseWriter, r *http.Request) {
	data, filename, timezone, ok := readExcelUpload(w, r)
	if !ok {
		return
	}
	result, err := s.Services.ExcelImports.Preview(data, timezone, r.FormValue("employeeQuery"), filename)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) importExcel(w http.ResponseWriter, r *http.Request) {
	data, filename, timezone, ok := readExcelUpload(w, r)
	if !ok {
		return
	}
	calendarID, _ := strconv.ParseInt(r.FormValue("calendarId"), 10, 64)
	result, err := s.Services.ExcelImports.Import(data, application.ExcelImportOptions{
		AllDay:        r.FormValue("allDay") == "true",
		CalendarID:    calendarID,
		EmployeeQuery: r.FormValue("employeeQuery"),
		SourceName:    filename,
		Timezone:      timezone,
		WorkEnd:       r.FormValue("workEnd"),
		WorkStart:     r.FormValue("workStart"),
	}, CurrentUser(r), clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func readExcelUpload(w http.ResponseWriter, r *http.Request) ([]byte, string, string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, application.MaxExcelImportBytes()+1024)
	if err := r.ParseMultipartForm(application.MaxExcelImportBytes()); err != nil {
		writeError(w, application.NewError("invalid_excel_import", "Excel-Datei konnte nicht gelesen werden", nil))
		return nil, "", "", false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, application.NewError("invalid_excel_import", "Excel-Datei fehlt", nil))
		return nil, "", "", false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, application.MaxExcelImportBytes()+1))
	if err != nil || int64(len(data)) > application.MaxExcelImportBytes() {
		writeError(w, application.NewError("invalid_excel_import", "Excel-Datei ist zu groß", nil))
		return nil, "", "", false
	}
	filename := ""
	if header != nil {
		filename = header.Filename
	}
	return data, filename, r.FormValue("timezone"), true
}
