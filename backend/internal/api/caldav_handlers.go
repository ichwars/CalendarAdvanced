package api

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/caldav"
	"calendaradvanced/internal/infrastructure/sqlite"
)

func (s *Server) caldavWellKnown(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/dav/", http.StatusPermanentRedirect)
}

func (s *Server) caldav(w http.ResponseWriter, r *http.Request) {
	user, ok := s.caldavAuth(w, r)
	if !ok {
		return
	}
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	w.Header().Set("DAV", "1, calendar-access")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	if r.Method == "PROPFIND" {
		if len(parts) <= 1 || path == "dav" {
			_, _ = w.Write([]byte(caldav.PrincipalResponse(s.Services.Config.PublicURL, user.Email)))
			return
		}
		if len(parts) >= 2 && parts[1] == "calendars" {
			calendars, _ := s.Services.Store.ListCalendars(user.ID)
			_, _ = w.Write([]byte(caldav.CalendarHomeResponse(user.Email, calendars)))
			return
		}
	}
	if r.Method == "REPORT" && len(parts) >= 4 && parts[1] == "calendars" {
		calendarID, _ := strconv.ParseInt(parts[3], 10, 64)
		events, err := s.Services.Store.ListEvents(sqlite.EventFilter{UserID: user.ID, CalendarID: calendarID, From: time.Now().AddDate(-1, 0, 0), To: time.Now().AddDate(1, 0, 0), Limit: 1000})
		if err != nil {
			http.Error(w, "calendar query failed", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(caldav.CalendarQueryResponse(user.Email, calendarID, events)))
		return
	}
	if r.Method == "GET" && strings.HasSuffix(path, ".ics") && len(parts) >= 5 {
		calendarID, _ := strconv.ParseInt(parts[3], 10, 64)
		events, err := s.Services.Store.ListEvents(sqlite.EventFilter{UserID: user.ID, CalendarID: calendarID, From: time.Now().AddDate(-1, 0, 0), To: time.Now().AddDate(1, 0, 0), Limit: 1000})
		if err != nil {
			http.Error(w, "calendar read failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		_, _ = w.Write([]byte(caldav.CalendarData(events)))
		return
	}
	http.Error(w, "CalDAV operation not implemented", http.StatusNotImplemented)
}

func (s *Server) caldavAuth(w http.ResponseWriter, r *http.Request) (user domain.User, ok bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Basic ") {
		w.Header().Set("WWW-Authenticate", `Basic realm="CalendarAdvanced CalDAV"`)
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return user, false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		http.Error(w, "invalid authentication", http.StatusUnauthorized)
		return user, false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		http.Error(w, "invalid authentication", http.StatusUnauthorized)
		return user, false
	}
	domainUser, err := s.Services.CalDAV.Authenticate(parts[0], parts[1])
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return user, false
	}
	return domainUser, true
}
