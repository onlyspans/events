package response

import (
	"errors"
	"net/http"

	"github.com/onlyspans/events/internal/apperr"
)

// ErrorBody represents a structured error response.
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Field   string `json:"field,omitempty"`
}

// Error writes an error response based on the error type.
// For AppError types, it uses the appropriate HTTP status and structured response.
// For other errors, it returns a 500 Internal Server Error with a generic message.
func Error(w http.ResponseWriter, err error) {
	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		body := ErrorBody{
			Error:   string(appErr.Type),
			Message: appErr.Message,
			Field:   appErr.Field,
		}
		JSON(w, appErr.HTTPStatus(), body)
		return
	}

	// For non-AppError types, return generic internal error
	body := ErrorBody{
		Error:   "internal",
		Message: "Internal server error",
	}
	JSON(w, http.StatusInternalServerError, body)
}

// BadRequest writes a 400 Bad Request error response.
func BadRequest(w http.ResponseWriter, message string) {
	body := ErrorBody{
		Error:   "validation",
		Message: message,
	}
	JSON(w, http.StatusBadRequest, body)
}

// NotFound writes a 404 Not Found error response.
func NotFound(w http.ResponseWriter, message string) {
	body := ErrorBody{
		Error:   "not_found",
		Message: message,
	}
	JSON(w, http.StatusNotFound, body)
}

// MethodNotAllowed writes a 405 Method Not Allowed error response.
func MethodNotAllowed(w http.ResponseWriter) {
	body := ErrorBody{
		Error:   "method_not_allowed",
		Message: "Method not allowed",
	}
	JSON(w, http.StatusMethodNotAllowed, body)
}

// InternalError writes a 500 Internal Server Error response.
func InternalError(w http.ResponseWriter, message string) {
	body := ErrorBody{
		Error:   "internal",
		Message: message,
	}
	JSON(w, http.StatusInternalServerError, body)
}

// ServiceUnavailable writes a 503 Service Unavailable error response.
func ServiceUnavailable(w http.ResponseWriter, message string) {
	body := ErrorBody{
		Error:   "service_unavailable",
		Message: message,
	}
	JSON(w, http.StatusServiceUnavailable, body)
}
