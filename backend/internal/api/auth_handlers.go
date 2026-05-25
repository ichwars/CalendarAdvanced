package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"calendaradvanced/internal/application"
)

func (s *Server) setupStatus(w http.ResponseWriter, r *http.Request) {
	required, err := s.Services.Setup.Status()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"required": required})
}

func (s *Server) setupAdmin(w http.ResponseWriter, r *http.Request) {
	if !s.Services.RateLimit.Allow("setup:"+clientIP(r), 5, time.Hour) {
		writeError(w, application.ErrRateLimited)
		return
	}
	var input application.SetupAdminInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	user, err := s.Services.Setup.CreateFirstAdmin(input, clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var input application.LoginInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	result, err := s.Services.Auth.Login(input, clientIP(r), userAgent(r))
	if err != nil {
		if errors.Is(err, application.ErrTwoFactorNeeded) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"twoFactorRequired": true, "code": "two_factor_required"})
			return
		}
		writeError(w, err)
		return
	}
	setSessionCookies(w, result.SessionToken, result.CSRFToken, s.Services.Config.CookieSecure, s.Services.Config.SessionTTL)
	writeJSON(w, http.StatusOK, map[string]any{"user": result.User, "csrfToken": result.CSRFToken, "twoFactorRequired": false})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie(sessionCookieName)
	var token string
	if cookie != nil {
		token = cookie.Value
	}
	_ = s.Services.Auth.Logout(token, clientIP(r), userAgent(r), CurrentUser(r).ID)
	clearCookies(w, s.Services.Config.CookieSecure)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"user": CurrentUser(r), "csrfToken": csrfFromCookie(r)})
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	if err := s.Services.Auth.ChangePassword(CurrentUser(r), input.CurrentPassword, input.NewPassword, clientIP(r), userAgent(r)); err != nil {
		writeError(w, err)
		return
	}
	clearCookies(w, s.Services.Config.CookieSecure)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	result, err := s.Services.Auth.RequestPasswordReset(input.Email, clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	if err := s.Services.Auth.ResetPassword(input.Token, input.NewPassword, clientIP(r), userAgent(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) twoFactorSetup(w http.ResponseWriter, r *http.Request) {
	secret, uri, err := s.Services.Auth.BeginTwoFactorSetup(CurrentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"secret": secret, "otpauthUri": uri})
}

func (s *Server) twoFactorEnable(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	codes, err := s.Services.Auth.EnableTwoFactor(CurrentUser(r), input.Code, clientIP(r), userAgent(r))
	if err != nil {
		writeError(w, err)
		return
	}
	clearCookies(w, s.Services.Config.CookieSecure)
	writeJSON(w, http.StatusOK, map[string]any{"backupCodes": codes})
}

func (s *Server) twoFactorDisable(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, application.ErrValidation)
		return
	}
	if err := s.Services.Auth.DisableTwoFactor(CurrentUser(r), input.Password, clientIP(r), userAgent(r)); err != nil {
		writeError(w, err)
		return
	}
	clearCookies(w, s.Services.Config.CookieSecure)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func setSessionCookies(w http.ResponseWriter, sessionToken, csrfToken string, secure bool, ttl time.Duration) {
	maxAge := int(ttl.Seconds())
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: sessionToken, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode, MaxAge: maxAge})
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: csrfToken, Path: "/", HttpOnly: false, Secure: secure, SameSite: http.SameSiteStrictMode, MaxAge: maxAge})
}

func clearCookies(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: "", Path: "/", HttpOnly: false, Secure: secure, SameSite: http.SameSiteStrictMode, MaxAge: -1})
}

func csrfFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func parseID(v string) int64 {
	id, _ := strconv.ParseInt(v, 10, 64)
	return id
}
