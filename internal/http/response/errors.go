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

	body := ErrorBody{
		Error:   "internal",
		Message: "Internal server error",
	}
	JSON(w, http.StatusInternalServerError, body)
}

func BadRequest(w http.ResponseWriter, message string) {
	body := ErrorBody{
		Error:   "validation",
		Message: message,
	}
	JSON(w, http.StatusBadRequest, body)
}

func NotFound(w http.ResponseWriter, message string) {
	body := ErrorBody{
		Error:   "not_found",
		Message: message,
	}
	JSON(w, http.StatusNotFound, body)
}

func MethodNotAllowed(w http.ResponseWriter) {
	body := ErrorBody{
		Error:   "method_not_allowed",
		Message: "Method not allowed",
	}
	JSON(w, http.StatusMethodNotAllowed, body)
}

func InternalError(w http.ResponseWriter, message string) {
	body := ErrorBody{
		Error:   "internal",
		Message: message,
	}
	JSON(w, http.StatusInternalServerError, body)
}

func ServiceUnavailable(w http.ResponseWriter, message string) {
	body := ErrorBody{
		Error:   "service_unavailable",
		Message: message,
	}
	JSON(w, http.StatusServiceUnavailable, body)
}
