package models

import "time"

// QuizContent defines the structure for a "quiz" content block.
type QuizContent struct {
	ID        int64      `json:"id" db:"id"`
	Title     string     `json:"title" db:"title"`
	Questions []Question `json:"questions"`
}

type Question struct {
	ID      string   `json:"id"`   // e.g., "q1", "q2"
	Type    string   `json:"type"` // "single_choice", "multiple_choice", "essay"
	Text    string   `json:"text"`
	Options []string `json:"options,omitempty"`
	Answer  any      `json:"-"` // Correct answer(s), omitted from client response
	Points  int      `json:"points"`
}

// QuizAttempt represents a user's submission for a quiz.
// This maps to the 'quiz_attempts' table.
type QuizAttempt struct {
	ID          int64          `json:"id" db:"id"`
	UserID      string         `json:"user_id" db:"user_id"`
	QuizID      int64          `json:"quiz_id" db:"quiz_id"`
	Answers     map[string]any `json:"answers" db:"answers"` // Stored as JSONB
	Score       int            `json:"score" db:"score"`
	Status      string         `json:"status" db:"status"` // "graded", "pending_manual_grade"
	SubmittedAt time.Time      `json:"submitted_at" db:"submitted_at"`
}

type QuizAttemptResult struct {
	AttemptID      int64          `json:"attempt_id"`
	Score          int            `json:"score"`
	TotalPoints    int            `json:"total_points"`
	Status         string         `json:"status"` // "graded", "pending_manual_grade"
	Feedback       string         `json:"feedback"`
	CorrectAnswers map[string]any `json:"correct_answers,omitempty"` // Show correct answers after submission
}

// SubmitQuizRequest defines the request body for submitting a quiz.
type SubmitQuizRequest struct {
	Answers map[string]any `json:"answers" validate:"required"` // Flexible map for quiz answers
}
