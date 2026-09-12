package errors

import (
	"encoding/json"
	"net/http"
)

type ErrorCode string

const (
	ErrInternal           ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrBadRequest         ErrorCode = "BAD_REQUEST"
	ErrNotFound           ErrorCode = "NOT_FOUND"
	ErrUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrForbidden          ErrorCode = "FORBIDDEN"
	ErrConflict           ErrorCode = "CONFLICT"
	ErrValidation         ErrorCode = "VALIDATION_FAILED"
	ErrTooManyRequests    ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

type APIError struct {
	HTTPStatus int       `json:"-"`
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    any       `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}

type ErrorResponse struct {
	Error *APIError `json:"error"`
}

func New(status int, code ErrorCode, message string) *APIError {
	return &APIError{
		HTTPStatus: status,
		Code:       code,
		Message:    message,
	}
}

func NewWithDetails(status int, code ErrorCode, message string, details any) *APIError {
	return &APIError{
		HTTPStatus: status,
		Code:       code,
		Message:    message,
		Details:    details,
	}
}

func NotFound(message string) *APIError {
	return New(http.StatusNotFound, ErrNotFound, message)
}

func BadRequest(message string) *APIError {
	return New(http.StatusBadRequest, ErrBadRequest, message)
}

func Unauthorized(message string) *APIError {
	return New(http.StatusUnauthorized, ErrUnauthorized, message)
}

func Forbidden(message string) *APIError {
	return New(http.StatusForbidden, ErrForbidden, message)
}

func Conflict(message string) *APIError {
	return New(http.StatusConflict, ErrConflict, message)
}

func Internal(message string) *APIError {
	return New(http.StatusInternalServerError, ErrInternal, message)
}

func RespondWithError(w http.ResponseWriter, err *APIError) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(err.HTTPStatus)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: err})
}
