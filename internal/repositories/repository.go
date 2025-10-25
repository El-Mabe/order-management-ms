package repositories

import "fmt"

type RepositoryError struct {
	StatusCode int    `json:"status_code"`
	Cause      string `json:"cause"`
	Message    string `json:"message"`
}

func (e *RepositoryError) Error() string {
	return fmt.Sprintf("status=%d, message=%s", e.StatusCode, e.Message)
}
