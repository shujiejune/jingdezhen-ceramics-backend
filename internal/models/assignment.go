package models

import "time"

type Assignment struct {
	ID             int64      `json:"id" db:"id"`
	Title          string     `json:"title" db:"title"`
	Description    string     `json:"description" db:"description"`
	AttachmentURLs []string   `json:"attachment_urls,omitempty" db:"attachment_urls"`
	ApplyDeadline  bool       `json:"apply_deadline" db:"apply_deadline"`
	Deadline       *time.Time `json:"deadline,omitempty" db:"deadline"`
}

// AssignmentSubmission represents a single attempt by a user for an assignment.
type AssignmentSubmission struct {
	ID           int64          `json:"id" db:"id"`
	AssignmentID int64          `json:"assignment_id" db:"assignment_id"`
	UserID       string         `json:"user_id" db:"user_id"`
	Answers      map[string]any `json:"answers" db:"answers"` // Stored as JSONB
	Status       string         `json:"status" db:"status"`   // "graded", "ungraded"
	SubmittedAt  time.Time      `json:"submitted_at" db:"submitted_at"`
	// Add fields for grading later, e.g., Grade, GradedAt, GraderID, Feedback
}

// SubmitAssignmentRequest defines the request body for submitting an assignment.
type SubmitAssignmentRequest struct {
	Answers map[string]any `json:"answers" validate:"required"` // Flexible map for assignment answers
}
