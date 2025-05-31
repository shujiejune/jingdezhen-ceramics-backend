package models

// ErrorResponse is a generic structure for JSON error responses.
type ErrorResponse struct {
	Message string `json:"message"`
	Details string `json:"details,omitempty"` // Optional additional details
}
