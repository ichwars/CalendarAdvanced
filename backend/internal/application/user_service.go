package application

import (
	"fmt"
	"strings"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/crypto"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type UserService struct {
	Store *sqlite.Store
	Audit *AuditService
}

type CreateUserInput struct {
	Email       string            `json:"email"`
	Username    string            `json:"username"`
	DisplayName string            `json:"displayName"`
	Password    string            `json:"password"`
	Roles       []domain.RoleName `json:"roles"`
}

func (s *UserService) List() ([]domain.User, error) {
	return s.Store.ListUsers()
}

func (s *UserService) Create(input CreateUserInput, actor domain.User, ip, userAgent string) (domain.User, error) {
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
	roles := normalizeRoles(input.Roles)
	if len(roles) == 0 {
		roles = []domain.RoleName{domain.RoleViewer}
	}
	hash, err := crypto.HashPassword(input.Password)
	if err != nil {
		return domain.User{}, err
	}
	user, err := s.Store.CreateUser(domain.NormalizeEmail(input.Email), domain.NormalizeUsername(input.Username), strings.TrimSpace(input.DisplayName), hash, true, roles)
	if err != nil {
		return domain.User{}, err
	}
	s.Audit.Record(actor.ID, domain.AuditUserCreated, "user", fmt.Sprint(user.ID), ip, userAgent, nil)
	return user, nil
}

func (s *UserService) SetRoles(userID int64, roles []domain.RoleName, actor domain.User, ip, userAgent string) error {
	roles = normalizeRoles(roles)
	if len(roles) == 0 {
		return NewError("roles_required", "at least one role is required", nil)
	}
	if err := s.Store.SetUserRoles(userID, roles); err != nil {
		return err
	}
	s.Audit.Record(actor.ID, domain.AuditUserUpdated, "user", fmt.Sprint(userID), ip, userAgent, map[string]any{"roles": roles})
	return nil
}

func normalizeRoles(in []domain.RoleName) []domain.RoleName {
	seen := map[domain.RoleName]bool{}
	out := make([]domain.RoleName, 0, len(in))
	for _, role := range in {
		switch role {
		case domain.RoleAdmin, domain.RoleEditor, domain.RoleViewer:
			if !seen[role] {
				seen[role] = true
				out = append(out, role)
			}
		}
	}
	return out
}
