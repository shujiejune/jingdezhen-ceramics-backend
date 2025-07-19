package models

import "time"

// Course represents a top-level online course.
type Course struct {
	ID           int64     `json:"id" db:"id"`
	Title        string    `json:"title" db:"title"`
	Description  string    `json:"description" db:"description"`
	InstructorID *string   `json:"instructor_id,omitempty" db:"instructor_id"` // User ID (UUID)
	ThumbnailURL string    `json:"thumbnail_url" db:"thumbnail_url"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	// This field will be populated by the service layer
	Chapters []CourseChapter `json:"chapters,omitempty" db:"-"`
}

// CourseChapter represents a single chapter within a course.
type CourseChapter struct {
	ID             int64  `json:"id" db:"id"`
	CourseID       int64  `json:"course_id" db:"course_id"`
	Title          string `json:"title" db:"title"`
	DisplayOrder   int    `json:"display_order" db:"display_order"`
	VideoURL       string `json:"video_url,omitempty" db:"video_url"`   // Omitted for non-enrolled users on protected chapters
	Content        string `json:"content,omitempty" db:"content"`       // Omitted for non-enrolled users on protected chapters
	ContentPreview string `json:"content_preview" db:"content_preview"` // Always available
	// User-specific progress, populated by the service layer
	ProgressPercentage int `json:"progress_percentage" db:"-"`
}

// UserChapterProgress tracks a specific user's progress in a chapter.
type UserChapterProgress struct {
	UserID             string     `json:"user_id" db:"user_id"`
	ChapterID          int64      `json:"chapter_id" db:"chapter_id"`
	ProgressPercentage int        `json:"progress_percentage" db:"progress_percentage"`
	VideoLastStoppedAt int        `json:"video_last_stopped_at" db:"video_last_stopped_at"` // in seconds
	CompletedAt        *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// UpdateProgressRequest defines the request body for updating chapter progress.
type UpdateProgressRequest struct {
	ProgressPercentage int `json:"progress_percentage" validate:"gte=0,lte=100"`
	VideoLastStoppedAt int `json:"video_last_stopped_at" validate:"gte=0"`
}

// SubmitQuizRequest defines the request body for submitting a quiz.
type SubmitQuizRequest struct {
	Answers map[string]interface{} `json:"answers" validate:"required"` // Flexible map for quiz answers
}
