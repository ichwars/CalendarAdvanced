package domain

import (
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode"
)

type RoleName string

const (
	RoleAdmin  RoleName = "admin"
	RoleEditor RoleName = "editor"
	RoleViewer RoleName = "viewer"
)

type User struct {
	ID               int64      `json:"id"`
	Email            string     `json:"email"`
	Username         string     `json:"username"`
	DisplayName      string     `json:"displayName"`
	PasswordHash     string     `json:"-"`
	Active           bool       `json:"active"`
	Roles            []RoleName `json:"roles"`
	TwoFactorEnabled bool       `json:"twoFactorEnabled"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type Session struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	TokenHash string    `json:"-"`
	CSRFHash  string    `json:"-"`
	ExpiresAt time.Time `json:"expiresAt"`
	RevokedAt time.Time `json:"revokedAt,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UserAgent string    `json:"userAgent,omitempty"`
	IP        string    `json:"ip,omitempty"`
}

func NormalizeEmail(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func NormalizeUsername(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func ValidateEmail(v string) error {
	v = NormalizeEmail(v)
	if len(v) < 6 || len(v) > 254 {
		return errors.New("email length is invalid")
	}
	addr, err := mail.ParseAddress(v)
	if err != nil || addr.Address != v {
		return errors.New("email format is invalid")
	}
	return nil
}

func ValidateDisplayName(v string) error {
	v = strings.TrimSpace(v)
	if len(v) < 2 || len(v) > 120 {
		return errors.New("display name length is invalid")
	}
	return nil
}

func ValidateUsername(v string) error {
	v = NormalizeUsername(v)
	if len(v) < 3 || len(v) > 40 {
		return errors.New("username must be between 3 and 40 characters")
	}
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return errors.New("username may contain lowercase letters, digits, dots, underscores and hyphens")
	}
	return nil
}

func ValidatePassword(v string) error {
	if len(v) < 12 || len(v) > 256 {
		return errors.New("password must be between 12 and 256 characters")
	}
	var hasLower, hasUpper, hasDigit, hasSymbol bool
	for _, r := range v {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.IsSpace(r):
			hasSymbol = true
		}
	}
	if !(hasLower && hasUpper && hasDigit && hasSymbol) {
		return errors.New("password must contain lowercase, uppercase, digit and symbol characters")
	}
	return nil
}

func HasRole(roles []RoleName, wanted RoleName) bool {
	for _, role := range roles {
		if role == wanted {
			return true
		}
	}
	return false
}
