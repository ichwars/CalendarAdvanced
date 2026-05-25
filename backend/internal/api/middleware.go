package api

import (
	"context"
	"net"
	"net/http"
	"strings"

	"calendaradvanced/internal/domain"
)

type contextKey string

const (
	userContextKey    contextKey = "user"
	sessionContextKey contextKey = "session"
)

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, ErrAuth)
			return
		}
		session, user, err := s.Services.Auth.Authenticate(cookie.Value)
		if err != nil {
			writeError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, sessionContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (s *Server) requireRole(role domain.RoleName, next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		user := CurrentUser(r)
		if !domain.HasRole(user.Roles, role) {
			writeError(w, ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireWrite(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			contentType := r.Header.Get("Content-Type")
			if r.Body != nil && !strings.HasPrefix(contentType, "application/json") && !strings.HasPrefix(contentType, "multipart/form-data") {
				writeError(w, ErrUnsupportedContentType)
				return
			}
		}
		session := CurrentSession(r)
		csrf := r.Header.Get("X-CSRF-Token")
		if csrf == "" {
			if cookie, err := r.Cookie(csrfCookieName); err == nil {
				csrf = cookie.Value
			}
		}
		if err := s.Services.Auth.ValidateCSRF(session, csrf); err != nil {
			writeError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireRoleWrite(role domain.RoleName, next http.HandlerFunc) http.HandlerFunc {
	return s.requireRole(role, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			contentType := r.Header.Get("Content-Type")
			if r.Body != nil && !strings.HasPrefix(contentType, "application/json") && !strings.HasPrefix(contentType, "multipart/form-data") {
				writeError(w, ErrUnsupportedContentType)
				return
			}
		}
		session := CurrentSession(r)
		csrf := r.Header.Get("X-CSRF-Token")
		if csrf == "" {
			if cookie, err := r.Cookie(csrfCookieName); err == nil {
				csrf = cookie.Value
			}
		}
		if err := s.Services.Auth.ValidateCSRF(session, csrf); err != nil {
			writeError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CurrentUser(r *http.Request) domain.User {
	user, _ := r.Context().Value(userContextKey).(domain.User)
	return user
}

func CurrentSession(r *http.Request) domain.Session {
	session, _ := r.Context().Value(sessionContextKey).(domain.Session)
	return session
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func userAgent(r *http.Request) string {
	return r.UserAgent()
}
