package errors

import (
	"net/http"
)

type ServiceError struct {
	Status            int           `json:"status"`
	Message           string        `json:"message"`
	Cause             []interface{} `json:"cause,omitempty"`
	StatusDescription string        `json:"status_description,omitempty"`
}

// Implementa la interfaz error
func (e *ServiceError) Error() string {
	return e.Message
}

// Constructor base
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

// Helpers convenientes
func BadRequest(message string, cause error) *ServiceError {
	return NewServiceError(http.StatusBadRequest, message, cause)
}

func NotFound(message string, cause error) *ServiceError {
	return NewServiceError(http.StatusNotFound, message, cause)
}

func Internal(message string, cause error) *ServiceError {
	return NewServiceError(http.StatusInternalServerError, message, cause)
}
