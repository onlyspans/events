// Package apperr provides structured application error types for consistent
// error handling across all layers of the application.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorType represents the category of an application error.
type ErrorType string

const (
	TypeValidation ErrorType = "validation"
	TypeNotFound   ErrorType = "not_found"
	TypeInternal   ErrorType = "internal"
	TypeConflict   ErrorType = "conflict"
)

// AppError is a structured error type that carries additional context
// for proper error handling and HTTP response generation.
type AppError struct {
	Type    ErrorType
	Message string
	Field   string // Optional: specific field that caused the error
	Err     error  // Optional: underlying error for wrapping
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatus returns the appropriate HTTP status code for this error type.
func (e *AppError) HTTPStatus() int {
	switch e.Type {
	case TypeValidation:
		return http.StatusBadRequest
	case TypeNotFound:
		return http.StatusNotFound
	case TypeConflict:
		return http.StatusConflict
	case TypeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// NewValidationError creates a validation error for invalid input.
func NewValidationError(message string) *AppError {
	return &AppError{
		Type:    TypeValidation,
		Message: message,
	}
}

// NewValidationErrorf creates a validation error with formatted message.
func NewValidationErrorf(format string, args ...any) *AppError {
	return &AppError{
		Type:    TypeValidation,
		Message: fmt.Sprintf(format, args...),
	}
}

// NewFieldValidationError creates a validation error for a specific field.
func NewFieldValidationError(field, message string) *AppError {
	return &AppError{
		Type:    TypeValidation,
		Message: message,
		Field:   field,
	}
}

// NewNotFoundError creates an error indicating a resource was not found.
func NewNotFoundError(message string) *AppError {
	return &AppError{
		Type:    TypeNotFound,
		Message: message,
	}
}

// NewNotFoundErrorf creates a not found error with formatted message.
func NewNotFoundErrorf(format string, args ...any) *AppError {
	return &AppError{
		Type:    TypeNotFound,
		Message: fmt.Sprintf(format, args...),
	}
}

// NewInternalError creates an internal error, typically for unexpected failures.
func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Type:    TypeInternal,
		Message: message,
		Err:     err,
	}
}

// NewInternalErrorf creates an internal error with formatted message.
func NewInternalErrorf(err error, format string, args ...any) *AppError {
	return &AppError{
		Type:    TypeInternal,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

// NewConflictError creates an error indicating a resource conflict.
func NewConflictError(message string) *AppError {
	return &AppError{
		Type:    TypeConflict,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, message string) *AppError {
	if err == nil {
		return nil
	}

	// If already an AppError, preserve the type but add context
	var appErr *AppError
	if errors.As(err, &appErr) {
		return &AppError{
			Type:    appErr.Type,
			Message: message + ": " + appErr.Message,
			Field:   appErr.Field,
			Err:     err,
		}
	}

	// Default to internal error for unknown errors
	return &AppError{
		Type:    TypeInternal,
		Message: message,
		Err:     err,
	}
}

// IsValidation checks if an error is a validation error.
func IsValidation(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == TypeValidation
}

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == TypeNotFound
}

// IsInternal checks if an error is an internal error.
func IsInternal(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == TypeInternal
}

// IsConflict checks if an error is a conflict error.
func IsConflict(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == TypeConflict
}

// GetHTTPStatus returns the HTTP status code for an error.
// Returns 500 for non-AppError types.
func GetHTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.HTTPStatus()
	}
	return http.StatusInternalServerError
}
