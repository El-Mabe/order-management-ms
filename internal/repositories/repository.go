package repositories

type RepositoryError struct {
	StatusCode int    `json:"status_code"`
	Cause      string `json:"cause"`
	Message    string `json:"message"`
}
