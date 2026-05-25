package application

import (
	"fmt"
	"strings"
	"time"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/config"
	"calendaradvanced/internal/infrastructure/crypto"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type AuthService struct {
	Config    config.Config
	Store     *sqlite.Store
	Audit     *AuditService
	RateLimit *RateLimitService
	Cipher    *crypto.TokenCipher
}

type LoginInput struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	TOTPCode   string `json:"totpCode"`
	BackupCode string `json:"backupCode"`
}

type LoginResult struct {
	User              domain.User `json:"user,omitempty"`
	SessionToken      string      `json:"-"`
	CSRFToken         string      `json:"csrfToken,omitempty"`
	TwoFactorRequired bool        `json:"twoFactorRequired"`
}

type ResetRequestResult struct {
	Sent       bool   `json:"sent"`
	LocalToken string `json:"localToken,omitempty"`
}

func (s *AuthService) Login(input LoginInput, ip, userAgent string) (LoginResult, error) {
	identifier := strings.TrimSpace(input.Email)
	key := "login:" + ip + ":" + strings.ToLower(identifier)
	if !s.RateLimit.Allow(key, 8, 15*time.Minute) {
		return LoginResult{}, ErrRateLimited
	}
	user, err := s.Store.FindUserByLogin(identifier)
	if err != nil || !user.Active || !crypto.VerifyPassword(user.PasswordHash, input.Password) {
		s.Audit.Record(0, domain.AuditLoginFailed, "user", strings.ToLower(identifier), ip, userAgent, nil)
		return LoginResult{}, ErrUnauthorized
	}
	if user.TwoFactorEnabled {
		secret, enabled, err := s.twoFactorSecret(user.ID)
		if err != nil || !enabled {
			return LoginResult{}, ErrUnauthorized
		}
		ok := false
		if input.TOTPCode != "" {
			ok = crypto.VerifyTOTP(secret, input.TOTPCode, time.Now().UTC())
		}
		if !ok && input.BackupCode != "" {
			used, err := s.Store.UseBackupCode(user.ID, crypto.HashToken(strings.TrimSpace(input.BackupCode)))
			if err != nil {
				return LoginResult{}, err
			}
			if used {
				ok = true
				s.Audit.Record(user.ID, domain.AuditTwoFactorRecoveryUsed, "user", fmt.Sprint(user.ID), ip, userAgent, nil)
			}
		}
		if !ok {
			return LoginResult{TwoFactorRequired: true}, ErrTwoFactorNeeded
		}
	}
	result, err := s.createSession(user, ip, userAgent)
	if err != nil {
		return LoginResult{}, err
	}
	s.RateLimit.Reset(key)
	s.Audit.Record(user.ID, domain.AuditLoginSucceeded, "user", fmt.Sprint(user.ID), ip, userAgent, nil)
	return result, nil
}

func (s *AuthService) createSession(user domain.User, ip, userAgent string) (LoginResult, error) {
	token, err := crypto.RandomToken(32)
	if err != nil {
		return LoginResult{}, err
	}
	csrfToken, err := crypto.RandomToken(32)
	if err != nil {
		return LoginResult{}, err
	}
	expires := time.Now().UTC().Add(s.Config.SessionTTL)
	if err := s.Store.CreateSession(user.ID, crypto.HashToken(token), crypto.HashToken(csrfToken), expires, userAgent, ip); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: user, SessionToken: token, CSRFToken: csrfToken}, nil
}

func (s *AuthService) Authenticate(token string) (domain.Session, domain.User, error) {
	if token == "" {
		return domain.Session{}, domain.User{}, ErrUnauthorized
	}
	session, user, err := s.Store.FindSession(crypto.HashToken(token))
	if err != nil || !user.Active {
		return domain.Session{}, domain.User{}, ErrUnauthorized
	}
	return session, user, nil
}

func (s *AuthService) ValidateCSRF(session domain.Session, csrfToken string) error {
	if csrfToken == "" || !crypto.EqualHash(session.CSRFHash, crypto.HashToken(csrfToken)) {
		return ErrForbidden
	}
	return nil
}

func (s *AuthService) Logout(token, ip, userAgent string, userID int64) error {
	if token != "" {
		_ = s.Store.RevokeSession(crypto.HashToken(token))
	}
	s.Audit.Record(userID, domain.AuditLogout, "session", "", ip, userAgent, nil)
	return nil
}

func (s *AuthService) ChangePassword(user domain.User, currentPassword, newPassword, ip, userAgent string) error {
	if !crypto.VerifyPassword(user.PasswordHash, currentPassword) {
		return ErrUnauthorized
	}
	if err := domain.ValidatePassword(newPassword); err != nil {
		return NewError("weak_password", err.Error(), nil)
	}
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.Store.UpdateUserPassword(user.ID, hash); err != nil {
		return err
	}
	s.Audit.Record(user.ID, domain.AuditPasswordChanged, "user", fmt.Sprint(user.ID), ip, userAgent, nil)
	return nil
}

func (s *AuthService) RequestPasswordReset(email, ip, userAgent string) (ResetRequestResult, error) {
	key := "reset:" + ip + ":" + domain.NormalizeEmail(email)
	if !s.RateLimit.Allow(key, 5, time.Hour) {
		return ResetRequestResult{}, ErrRateLimited
	}
	user, err := s.Store.FindUserByEmail(email)
	if err != nil || !user.Active {
		return ResetRequestResult{Sent: true}, nil
	}
	token, err := crypto.RandomToken(32)
	if err != nil {
		return ResetRequestResult{}, err
	}
	if err := s.Store.InsertPasswordResetToken(user.ID, crypto.HashToken(token), time.Now().UTC().Add(30*time.Minute)); err != nil {
		return ResetRequestResult{}, err
	}
	s.Audit.Record(user.ID, domain.AuditPasswordResetRequested, "user", fmt.Sprint(user.ID), ip, userAgent, nil)
	result := ResetRequestResult{Sent: true}
	if s.Config.LocalResetTokens && s.Config.SMTP.Host == "" {
		result.LocalToken = token
	}
	return result, nil
}

func (s *AuthService) ResetPassword(token, newPassword, ip, userAgent string) error {
	if err := domain.ValidatePassword(newPassword); err != nil {
		return NewError("weak_password", err.Error(), nil)
	}
	hashToken := crypto.HashToken(token)
	userID, err := s.Store.FindPasswordResetToken(hashToken)
	if err != nil {
		return ErrUnauthorized
	}
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.Store.UpdateUserPassword(userID, hash); err != nil {
		return err
	}
	_ = s.Store.MarkPasswordResetUsed(hashToken)
	s.Audit.Record(userID, domain.AuditPasswordResetCompleted, "user", fmt.Sprint(userID), ip, userAgent, nil)
	return nil
}

func (s *AuthService) BeginTwoFactorSetup(user domain.User) (string, string, error) {
	secret, err := crypto.NewTOTPSecret()
	if err != nil {
		return "", "", err
	}
	storedSecret := secret
	if s.Cipher != nil {
		encrypted, err := s.Cipher.Encrypt(secret)
		if err != nil {
			return "", "", err
		}
		storedSecret = "v1:" + encrypted
	}
	if err := s.Store.UpsertTwoFactorSecret(user.ID, storedSecret, false); err != nil {
		return "", "", err
	}
	issuer := "CalendarAdvanced"
	uri := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30", issuer, user.Email, secret, issuer)
	return secret, uri, nil
}

func (s *AuthService) EnableTwoFactor(user domain.User, code, ip, userAgent string) ([]string, error) {
	secret, _, err := s.twoFactorSecret(user.ID)
	if err != nil || !crypto.VerifyTOTP(secret, code, time.Now().UTC()) {
		return nil, ErrUnauthorized
	}
	codes := make([]string, 10)
	hashes := make([]string, 10)
	for i := range codes {
		code, err := crypto.RandomToken(10)
		if err != nil {
			return nil, err
		}
		codes[i] = code
		hashes[i] = crypto.HashToken(code)
	}
	if err := s.Store.ReplaceBackupCodes(user.ID, hashes); err != nil {
		return nil, err
	}
	if err := s.Store.SetTwoFactorEnabled(user.ID, true); err != nil {
		return nil, err
	}
	_ = s.Store.RevokeUserSessions(user.ID)
	s.Audit.Record(user.ID, domain.AuditTwoFactorEnabled, "user", fmt.Sprint(user.ID), ip, userAgent, nil)
	return codes, nil
}

func (s *AuthService) DisableTwoFactor(user domain.User, password, ip, userAgent string) error {
	if !crypto.VerifyPassword(user.PasswordHash, password) {
		return ErrUnauthorized
	}
	if err := s.Store.SetTwoFactorEnabled(user.ID, false); err != nil {
		return err
	}
	_ = s.Store.ReplaceBackupCodes(user.ID, nil)
	_ = s.Store.RevokeUserSessions(user.ID)
	s.Audit.Record(user.ID, domain.AuditTwoFactorDisabled, "user", fmt.Sprint(user.ID), ip, userAgent, nil)
	return nil
}

func (s *AuthService) twoFactorSecret(userID int64) (string, bool, error) {
	stored, enabled, err := s.Store.GetTwoFactorSecret(userID)
	if err != nil {
		return "", false, err
	}
	if strings.HasPrefix(stored, "v1:") {
		if s.Cipher == nil {
			return "", false, NewError("token_encryption_not_configured", "token encryption key is required to read encrypted 2FA secrets", nil)
		}
		plain, err := s.Cipher.Decrypt(strings.TrimPrefix(stored, "v1:"))
		if err != nil {
			return "", false, err
		}
		return plain, enabled, nil
	}
	return stored, enabled, nil
}
