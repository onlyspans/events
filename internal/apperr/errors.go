package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

type ErrorType string

const (
	TypeValidation ErrorType = "validation"
	TypeNotFound   ErrorType = "not_found"
	TypeInternal   ErrorType = "internal"
	TypeConflict   ErrorType = "conflict"
)

type AppError struct {
	Type    ErrorType
	Message string
	Field   string
	Err     error
}

func (e *AppError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

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

func NewValidationError(message string) *AppError {
	return &AppError{
		Type:    TypeValidation,
		Message: message,
	}
}

func NewValidationErrorf(format string, args ...any) *AppError {
	return &AppError{
		Type:    TypeValidation,
		Message: fmt.Sprintf(format, args...),
	}
}

func NewFieldValidationError(field, message string) *AppError {
	return &AppError{
		Type:    TypeValidation,
		Message: message,
		Field:   field,
	}
}

func NewNotFoundError(message string) *AppError {
	return &AppError{
		Type:    TypeNotFound,
		Message: message,
	}
}

func NewNotFoundErrorf(format string, args ...any) *AppError {
	return &AppError{
		Type:    TypeNotFound,
		Message: fmt.Sprintf(format, args...),
	}
}

func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Type:    TypeInternal,
		Message: message,
		Err:     err,
	}
}

func NewInternalErrorf(err error, format string, args ...any) *AppError {
	return &AppError{
		Type:    TypeInternal,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

func NewConflictError(message string) *AppError {
	return &AppError{
		Type:    TypeConflict,
		Message: message,
	}
}

func Wrap(err error, message string) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return &AppError{
			Type:    appErr.Type,
			Message: message + ": " + appErr.Message,
			Field:   appErr.Field,
			Err:     err,
		}
	}

	return &AppError{
		Type:    TypeInternal,
		Message: message,
		Err:     err,
	}
}

func IsValidation(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == TypeValidation
}

func IsNotFound(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == TypeNotFound
}

func IsInternal(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == TypeInternal
}

func IsConflict(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == TypeConflict
}

func GetHTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.HTTPStatus()
	}
	return http.StatusInternalServerError
}
