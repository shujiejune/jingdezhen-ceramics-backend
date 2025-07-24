package models

import "time"

// Course represents a top-level online course.
type Course struct {
	ID           int64           `json:"id" db:"id"`
	Title        string          `json:"title" db:"title"`
	Description  string          `json:"description" db:"description"`
	InstructorID *string         `json:"instructor_id,omitempty" db:"instructor_id"` // User ID (UUID)
	ThumbnailURL string          `json:"thumbnail_url" db:"thumbnail_url"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
	Chapters     []CourseChapter `json:"chapters,omitempty" db:"-"`
}

// CourseChapter represents a single chapter within a course.
type CourseChapter struct {
	ID                 int64  `json:"id" db:"id"`
	CourseID           int64  `json:"course_id" db:"course_id"`
	Title              string `json:"title" db:"title"`
	DisplayOrder       int    `json:"display_order" db:"display_order"`
	AvailableForGuests bool   `json:"available_to_guests" db:"available_to_guests"`
	// User-specific progress, populated by the service layer
	ProgressPercentage int                   `json:"progress_percentage" db:"-"`
	ResourceLinks      []string              `json:"resource_links,omitempty" db:"-"`
	ContentBlocks      []ChapterContentBlock `json:"content_blocks,omitempty" db:"-"`
}

// ChapterContentBlock represents a single piece of content within a chapter.
type ChapterContentBlock struct {
	ID           int64  `json:"id" db:"id"`
	ChapterID    int64  `json:"chapter_id" db:"chapter_id"`
	Type         string `json:"type" db:"type"`       // "video", "passage", "quiz"
	Content      any    `json:"content" db:"content"` // Stored as JSONB
	DisplayOrder int    `json:"display_order" db:"display_order"`
}

// --- Structs for the JSONB Content field ---

// VideoContent defines the structure for a "video" content block.
type VideoContent struct {
	URL      string `json:"url"`
	Duration int    `json:"duration_seconds"`
	Title    string `json:"title,omitempty"`
}

// PassageContent defines the structure for a "passage" content block.
type PassageContent struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// AssignmentContent defines the structure for an "assignment" content block.
type AssignmentContent struct {
	AssignmentID   int64      `json:"assignment_id"`
	Description    string     `json:"description"`
	AttachmentURLs []string   `json:"attachment_urls,omitempty"`
	ApplyDeadline  bool       `json:"apply_deadline"`
	Deadline       *time.Time `json:"deadline"`
}

// QuizContent and SubmitQuizRequest are defined in quiz.go

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

// SubmitAssignmentRequest defines the request body for submitting a quiz.
type SubmitAssignmentRequest struct {
	Answers map[string]any `json:"answers" validate:"required"` // Flexible map for assignment answers
}
