package errors

import (
	"net/http"
)

// ServiceError represents a standardized error structure for the application.
type ServiceError struct {
	Status            int           `json:"status"`
	Message           string        `json:"message"`
	Cause             []interface{} `json:"cause,omitempty"`
	StatusDescription string        `json:"status_description,omitempty"`
}

// Error implements the error interface for ServiceError.
func (e *ServiceError) Error() string {
	return e.Message
}

// NewServiceError creates a new ServiceError with the given HTTP status, message, and optional cause.
func NewServiceError(status int, message string, cause error) *ServiceError {
	causes := []interface{}{}
	if cause != nil {
		causes = append(causes, cause.Error())
	}

	return &ServiceError{
		Status:            status,
		Message:           message,
		Cause:             causes,
		StatusDescription: http.StatusText(status),
	}
}

// BadRequest returns a ServiceError representing HTTP 400 Bad Request.
func BadRequest(message string, cause error) *ServiceError {
	return NewServiceError(http.StatusBadRequest, message, cause)
}

// NotFound returns a ServiceError representing HTTP 404 Not Found.
func NotFound(message string, cause error) *ServiceError {
	return NewServiceError(http.StatusNotFound, message, cause)
}

// Internal returns a ServiceError representing HTTP 500 Internal Server Error.
func Internal(message string, cause error) *ServiceError {
	return NewServiceError(http.StatusInternalServerError, message, cause)
}
