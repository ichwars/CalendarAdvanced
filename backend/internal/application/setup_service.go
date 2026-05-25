package application

import (
	"fmt"
	"strings"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/crypto"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type SetupService struct {
	Store *sqlite.Store
	Audit *AuditService
}

type SetupAdminInput struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

func (s *SetupService) Status() (bool, error) {
	return s.Store.SetupRequired()
}

func (s *SetupService) CreateFirstAdmin(input SetupAdminInput, ip, userAgent string) (domain.User, error) {
	required, err := s.Store.SetupRequired()
	if err != nil {
		return domain.User{}, err
	}
	if !required {
		return domain.User{}, ErrSetupNotRequired
	}
	if err := domain.ValidateEmail(input.Email); err != nil {
		return domain.User{}, NewError("invalid_email", err.Error(), nil)
	}
	if err := domain.ValidateUsername(input.Username); err != nil {
		return domain.User{}, NewError("invalid_username", err.Error(), nil)
	}
	if err := domain.ValidateDisplayName(input.DisplayName); err != nil {
		return domain.User{}, NewError("invalid_display_name", err.Error(), nil)
	}
	if err := domain.ValidatePassword(input.Password); err != nil {
		return domain.User{}, NewError("weak_password", err.Error(), nil)
	}
	hash, err := crypto.HashPassword(input.Password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash setup password: %w", err)
	}
	user, err := s.Store.CreateUser(domain.NormalizeEmail(input.Email), domain.NormalizeUsername(input.Username), strings.TrimSpace(input.DisplayName), hash, true, []domain.RoleName{domain.RoleAdmin, domain.RoleEditor, domain.RoleViewer})
	if err != nil {
		return domain.User{}, err
	}
	s.Audit.Record(user.ID, domain.AuditSetupAdminCreated, "user", fmt.Sprint(user.ID), ip, userAgent, nil)
	return user, nil
}
