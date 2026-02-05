package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onlyspans/events/internal/apperr"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       any
		wantStatus int
		wantBody   string
	}{
		{
			name:       "with data",
			status:     http.StatusOK,
			data:       map[string]string{"key": "value"},
			wantStatus: http.StatusOK,
			wantBody:   `{"key":"value"}`,
		},
		{
			name:       "nil data",
			status:     http.StatusOK,
			data:       nil,
			wantStatus: http.StatusOK,
			wantBody:   "",
		},
		{
			name:       "custom status",
			status:     http.StatusAccepted,
			data:       map[string]int{"count": 42},
			wantStatus: http.StatusAccepted,
			wantBody:   `{"count":42}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			JSON(rec, tt.status, tt.data)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			if tt.data != nil {
				contentType := rec.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", contentType)
				}
			}

			if tt.wantBody != "" {
				body := rec.Body.String()
				// JSON encoder adds newline
				if body != tt.wantBody+"\n" {
					t.Errorf("expected body %q, got %q", tt.wantBody, body)
				}
			}
		})
	}
}

func TestOK(t *testing.T) {
	rec := httptest.NewRecorder()
	OK(rec, map[string]string{"status": "ok"})

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestCreated(t *testing.T) {
	rec := httptest.NewRecorder()
	Created(rec, map[string]string{"id": "123"})

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}
}

func TestNoContent(t *testing.T) {
	rec := httptest.NewRecorder()
	NoContent(rec)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}

	if rec.Body.Len() != 0 {
		t.Error("expected empty body")
	}
}

func TestMultiStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	MultiStatus(rec, map[string]int{"success": 5, "failed": 2})

	if rec.Code != http.StatusMultiStatus {
		t.Errorf("expected status 207, got %d", rec.Code)
	}
}

func TestError_WithAppError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantType   string
	}{
		{
			name:       "validation error",
			err:        apperr.NewValidationError("invalid input"),
			wantStatus: http.StatusBadRequest,
			wantType:   "validation",
		},
		{
			name:       "not found error",
			err:        apperr.NewNotFoundError("resource not found"),
			wantStatus: http.StatusNotFound,
			wantType:   "not_found",
		},
		{
			name:       "internal error",
			err:        apperr.NewInternalError("something went wrong", nil),
			wantStatus: http.StatusInternalServerError,
			wantType:   "internal",
		},
		{
			name:       "conflict error",
			err:        apperr.NewConflictError("resource already exists"),
			wantStatus: http.StatusConflict,
			wantType:   "conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Error(rec, tt.err)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			var body ErrorBody
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}

			if body.Error != tt.wantType {
				t.Errorf("expected error type %q, got %q", tt.wantType, body.Error)
			}
		})
	}
}

func TestError_WithFieldValidation(t *testing.T) {
	rec := httptest.NewRecorder()
	err := apperr.NewFieldValidationError("email", "invalid format")
	Error(rec, err)

	var body ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body.Field != "email" {
		t.Errorf("expected field 'email', got %q", body.Field)
	}
}

func TestError_WithGenericError(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, apperr.NewInternalError("db error", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	var body ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body.Error != "internal" {
		t.Errorf("expected error type 'internal', got %q", body.Error)
	}
}

func TestBadRequest(t *testing.T) {
	rec := httptest.NewRecorder()
	BadRequest(rec, "invalid parameter")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var body ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body.Message != "invalid parameter" {
		t.Errorf("expected message 'invalid parameter', got %q", body.Message)
	}
}

func TestNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	NotFound(rec, "event not found")

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	MethodNotAllowed(rec)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

func TestInternalError(t *testing.T) {
	rec := httptest.NewRecorder()
	InternalError(rec, "database error")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func TestServiceUnavailable(t *testing.T) {
	rec := httptest.NewRecorder()
	ServiceUnavailable(rec, "database unavailable")

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
}
