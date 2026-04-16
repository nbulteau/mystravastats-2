package domain

import (
	"errors"
	"fmt"
)

// DomainError est l'erreur de base pour tous les erreurs métier
type DomainError struct {
	Code    string
	Message string
	Cause   error
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

// Erreurs spécifiques
var (
	ErrNotFound       = errors.New("not_found")
	ErrValidation     = errors.New("validation_error")
	ErrRateLimited    = errors.New("rate_limited")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrConflict       = errors.New("conflict")
	ErrInternal       = errors.New("internal_error")
	ErrNotImplemented = errors.New("not_implemented")
)

// Factory functions
func NewNotFoundError(resource string, id interface{}) *DomainError {
	return &DomainError{
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s not found: %v", resource, id),
	}
}

func NewValidationError(field, message string) *DomainError {
	return &DomainError{
		Code:    "VALIDATION_ERROR",
		Message: fmt.Sprintf("invalid %s: %s", field, message),
	}
}

func NewRateLimitError(source string, retryAfter string) *DomainError {
	return &DomainError{
		Code:    "RATE_LIMITED",
		Message: fmt.Sprintf("%s rate limited. Retry after: %s", source, retryAfter),
	}
}

func NewUnauthorizedError(reason string) *DomainError {
	return &DomainError{
		Code:    "UNAUTHORIZED",
		Message: fmt.Sprintf("unauthorized: %s", reason),
	}
}

func NewConflictError(resource string, details string) *DomainError {
	return &DomainError{
		Code:    "CONFLICT",
		Message: fmt.Sprintf("conflict on %s: %s", resource, details),
	}
}

func NewInternalError(operation string, cause error) *DomainError {
	return &DomainError{
		Code:    "INTERNAL_ERROR",
		Message: fmt.Sprintf("internal error during %s", operation),
		Cause:   cause,
	}
}

// Helper para convertir en erreur HTTP
type HTTPError interface {
	error
	HTTPStatus() int
}

func (e *DomainError) HTTPStatus() int {
	switch e.Code {
	case "VALIDATION_ERROR":
		return 400
	case "UNAUTHORIZED":
		return 401
	case "NOT_FOUND":
		return 404
	case "CONFLICT":
		return 409
	case "RATE_LIMITED":
		return 429
	case "NOT_IMPLEMENTED":
		return 501
	default:
		return 500
	}
}

// Helpers para verificar tipo de erro
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var de *DomainError
	if errors.As(err, &de) {
		return de.Code == "NOT_FOUND"
	}
	return errors.Is(err, ErrNotFound)
}

func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	var de *DomainError
	if errors.As(err, &de) {
		return de.Code == "VALIDATION_ERROR"
	}
	return errors.Is(err, ErrValidation)
}

func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	var de *DomainError
	if errors.As(err, &de) {
		return de.Code == "RATE_LIMITED"
	}
	return errors.Is(err, ErrRateLimited)
}
