package application

import "errors"

var (
	ErrValidation       = errors.New("validation_failed")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not_found")
	ErrConflict         = errors.New("conflict")
	ErrRateLimited      = errors.New("rate_limited")
	ErrSetupNotRequired = errors.New("setup_not_required")
	ErrTwoFactorNeeded  = errors.New("two_factor_required")
)

type CodedError struct {
	Code    string
	Message string
	Details any
}

func (e CodedError) Error() string { return e.Code }

func NewError(code, message string, details any) CodedError {
	return CodedError{Code: code, Message: message, Details: details}
}
