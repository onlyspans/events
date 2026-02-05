package apperr

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name:     "simple message",
			err:      &AppError{Type: TypeValidation, Message: "invalid input"},
			expected: "invalid input",
		},
		{
			name:     "with field",
			err:      &AppError{Type: TypeValidation, Message: "is required", Field: "user_name"},
			expected: "user_name: is required",
		},
		{
			name:     "empty message",
			err:      &AppError{Type: TypeInternal, Message: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	underlying := errors.New("database connection failed")
	appErr := &AppError{
		Type:    TypeInternal,
		Message: "failed to save",
		Err:     underlying,
	}

	if got := appErr.Unwrap(); got != underlying {
		t.Errorf("Unwrap() = %v, want %v", got, underlying)
	}

	// Test with errors.Is
	if !errors.Is(appErr, underlying) {
		t.Error("errors.Is should find the underlying error")
	}
}

func TestAppError_HTTPStatus(t *testing.T) {
	tests := []struct {
		errType  ErrorType
		expected int
	}{
		{TypeValidation, http.StatusBadRequest},
		{TypeNotFound, http.StatusNotFound},
		{TypeConflict, http.StatusConflict},
		{TypeInternal, http.StatusInternalServerError},
		{ErrorType("unknown"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(string(tt.errType), func(t *testing.T) {
			err := &AppError{Type: tt.errType}
			if got := err.HTTPStatus(); got != tt.expected {
				t.Errorf("HTTPStatus() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("invalid email format")

	if err.Type != TypeValidation {
		t.Errorf("Type = %v, want %v", err.Type, TypeValidation)
	}
	if err.Message != "invalid email format" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid email format")
	}
	if err.HTTPStatus() != http.StatusBadRequest {
		t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), http.StatusBadRequest)
	}
}

func TestNewValidationErrorf(t *testing.T) {
	err := NewValidationErrorf("field %s must be at least %d characters", "name", 3)

	if err.Type != TypeValidation {
		t.Errorf("Type = %v, want %v", err.Type, TypeValidation)
	}
	expected := "field name must be at least 3 characters"
	if err.Message != expected {
		t.Errorf("Message = %q, want %q", err.Message, expected)
	}
}

func TestNewFieldValidationError(t *testing.T) {
	err := NewFieldValidationError("email", "must be a valid email address")

	if err.Type != TypeValidation {
		t.Errorf("Type = %v, want %v", err.Type, TypeValidation)
	}
	if err.Field != "email" {
		t.Errorf("Field = %q, want %q", err.Field, "email")
	}
	if err.Message != "must be a valid email address" {
		t.Errorf("Message = %q, want %q", err.Message, "must be a valid email address")
	}

	// Error string should include field
	expectedErr := "email: must be a valid email address"
	if err.Error() != expectedErr {
		t.Errorf("Error() = %q, want %q", err.Error(), expectedErr)
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("event not found")

	if err.Type != TypeNotFound {
		t.Errorf("Type = %v, want %v", err.Type, TypeNotFound)
	}
	if err.Message != "event not found" {
		t.Errorf("Message = %q, want %q", err.Message, "event not found")
	}
	if err.HTTPStatus() != http.StatusNotFound {
		t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), http.StatusNotFound)
	}
}

func TestNewNotFoundErrorf(t *testing.T) {
	err := NewNotFoundErrorf("event with ID %s not found", "abc-123")

	if err.Type != TypeNotFound {
		t.Errorf("Type = %v, want %v", err.Type, TypeNotFound)
	}
	expected := "event with ID abc-123 not found"
	if err.Message != expected {
		t.Errorf("Message = %q, want %q", err.Message, expected)
	}
}

func TestNewInternalError(t *testing.T) {
	underlying := errors.New("connection refused")
	err := NewInternalError("failed to connect to database", underlying)

	if err.Type != TypeInternal {
		t.Errorf("Type = %v, want %v", err.Type, TypeInternal)
	}
	if err.Message != "failed to connect to database" {
		t.Errorf("Message = %q, want %q", err.Message, "failed to connect to database")
	}
	if err.Err != underlying {
		t.Errorf("Err = %v, want %v", err.Err, underlying)
	}
	if err.HTTPStatus() != http.StatusInternalServerError {
		t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), http.StatusInternalServerError)
	}
}

func TestNewInternalErrorf(t *testing.T) {
	underlying := errors.New("timeout")
	err := NewInternalErrorf(underlying, "operation %s failed after %d retries", "save", 3)

	if err.Type != TypeInternal {
		t.Errorf("Type = %v, want %v", err.Type, TypeInternal)
	}
	expected := "operation save failed after 3 retries"
	if err.Message != expected {
		t.Errorf("Message = %q, want %q", err.Message, expected)
	}
	if err.Err != underlying {
		t.Errorf("Err = %v, want %v", err.Err, underlying)
	}
}

func TestNewConflictError(t *testing.T) {
	err := NewConflictError("resource already exists")

	if err.Type != TypeConflict {
		t.Errorf("Type = %v, want %v", err.Type, TypeConflict)
	}
	if err.Message != "resource already exists" {
		t.Errorf("Message = %q, want %q", err.Message, "resource already exists")
	}
	if err.HTTPStatus() != http.StatusConflict {
		t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), http.StatusConflict)
	}
}

func TestWrap(t *testing.T) {
	t.Run("wrap nil error", func(t *testing.T) {
		result := Wrap(nil, "context")
		if result != nil {
			t.Errorf("Wrap(nil) = %v, want nil", result)
		}
	})

	t.Run("wrap standard error", func(t *testing.T) {
		underlying := errors.New("original error")
		result := Wrap(underlying, "additional context")

		if result.Type != TypeInternal {
			t.Errorf("Type = %v, want %v", result.Type, TypeInternal)
		}
		if result.Message != "additional context" {
			t.Errorf("Message = %q, want %q", result.Message, "additional context")
		}
		if !errors.Is(result, underlying) {
			t.Error("wrapped error should contain original error")
		}
	})

	t.Run("wrap AppError preserves type", func(t *testing.T) {
		original := NewValidationError("invalid input")
		result := Wrap(original, "while processing request")

		if result.Type != TypeValidation {
			t.Errorf("Type = %v, want %v", result.Type, TypeValidation)
		}
		expectedMsg := "while processing request: invalid input"
		if result.Message != expectedMsg {
			t.Errorf("Message = %q, want %q", result.Message, expectedMsg)
		}
	})

	t.Run("wrap AppError preserves field", func(t *testing.T) {
		original := NewFieldValidationError("email", "invalid format")
		result := Wrap(original, "user creation failed")

		if result.Field != "email" {
			t.Errorf("Field = %q, want %q", result.Field, "email")
		}
	})
}

func TestIsValidation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"validation error", NewValidationError("invalid"), true},
		{"not found error", NewNotFoundError("missing"), false},
		{"internal error", NewInternalError("failed", nil), false},
		{"standard error", errors.New("some error"), false},
		{"wrapped validation", Wrap(NewValidationError("invalid"), "context"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidation(tt.err); got != tt.expected {
				t.Errorf("IsValidation() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"not found error", NewNotFoundError("missing"), true},
		{"validation error", NewValidationError("invalid"), false},
		{"standard error", errors.New("some error"), false},
		{"wrapped not found", Wrap(NewNotFoundError("missing"), "context"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.expected {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsInternal(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"internal error", NewInternalError("failed", nil), true},
		{"validation error", NewValidationError("invalid"), false},
		{"standard error", errors.New("some error"), false},
		{"wrapped internal", Wrap(NewInternalError("failed", nil), "context"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInternal(tt.err); got != tt.expected {
				t.Errorf("IsInternal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsConflict(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"conflict error", NewConflictError("exists"), true},
		{"validation error", NewValidationError("invalid"), false},
		{"standard error", errors.New("some error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConflict(tt.err); got != tt.expected {
				t.Errorf("IsConflict() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"validation error", NewValidationError("invalid"), http.StatusBadRequest},
		{"not found error", NewNotFoundError("missing"), http.StatusNotFound},
		{"internal error", NewInternalError("failed", nil), http.StatusInternalServerError},
		{"conflict error", NewConflictError("exists"), http.StatusConflict},
		{"standard error", errors.New("some error"), http.StatusInternalServerError},
		{"wrapped validation", Wrap(NewValidationError("invalid"), "ctx"), http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetHTTPStatus(tt.err); got != tt.expected {
				t.Errorf("GetHTTPStatus() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestErrorsAs(t *testing.T) {
	// Test that errors.As works correctly with AppError
	underlying := errors.New("db error")
	appErr := NewInternalError("failed to save", underlying)

	var target *AppError
	if !errors.As(appErr, &target) {
		t.Error("errors.As should succeed for AppError")
	}
	if target.Type != TypeInternal {
		t.Errorf("Type = %v, want %v", target.Type, TypeInternal)
	}
}

func TestErrorChaining(t *testing.T) {
	// Test that error chaining works correctly
	dbErr := errors.New("connection timeout")
	repoErr := NewInternalError("failed to query database", dbErr)
	serviceErr := Wrap(repoErr, "failed to fetch events")

	// Should be able to find original error
	if !errors.Is(serviceErr, dbErr) {
		t.Error("should be able to find original db error in chain")
	}

	// Should be able to extract AppError
	var appErr *AppError
	if !errors.As(serviceErr, &appErr) {
		t.Error("should be able to extract AppError from chain")
	}

	// Type should be preserved through wrapping
	if appErr.Type != TypeInternal {
		t.Errorf("Type = %v, want %v", appErr.Type, TypeInternal)
	}
}
