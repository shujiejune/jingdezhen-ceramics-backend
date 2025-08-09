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

// UserEnrollment represents the relationship between a user and a course.
type UserEnrollment struct {
	UserID        string    `json:"user_id" db:"user_id"`
	CourseID      int64     `json:"course_id" db:"course_id"`
	EnrolledAt    time.Time `json:"enrolled_at" db:"enrolled_at"`
	LastVisitedAt time.Time `json:"last_visited_at" db:"last_visited_at"`
}

// EnrolledCourseResponse is for listing enrolled courses by most recently visited order in user profile.
type EnrolledCourseResponse struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	ThumbnailURL  string    `json:"thumbnail_url"`
	LastVisitedAt time.Time `json:"last_visited_at"`
}

// CourseChapter represents a single chapter within a course.
type CourseChapter struct {
	ID                 int64                 `json:"id" db:"id"`
	CourseID           int64                 `json:"course_id" db:"course_id"`
	Title              string                `json:"title" db:"title"`
	DisplayOrder       int                   `json:"display_order" db:"display_order"`
	AvailableForGuests bool                  `json:"available_for_guests" db:"available_for_guests"`
	ProgressPercentage int                   `json:"progress_percentage" db:"-"`
	ResourceLinks      []string              `json:"resource_links,omitempty" db:"-"`
	ContentBlockCount  ChapterContentCount   `json:"content_block_count" db:"-"`
	ContentBlocks      []ChapterContentBlock `json:"content_blocks,omitempty" db:"-"`
}

// ChapterContentCount stores the number of each type of content block within a chapter.
type ChapterContentCount struct {
	VideoCount      int `json:"video_count" db:"video_count"`
	ReadingCount    int `json:"reading_count" db:"reading_count"`
	QuizCount       int `json:"quiz_count" db:"quiz_count"`
	AssignmentCount int `json:"assignment_count" db:"assignment_count"`
}

// ChapterContentBlock represents a single piece of content within a chapter.
type ChapterContentBlock struct {
	ID           int64     `json:"id" db:"id"`
	ChapterID    int64     `json:"chapter_id" db:"chapter_id"`
	CourseID     int64     `json:"course_id" db:"course_id"`
	Type         string    `json:"type" db:"type"`       // "video", "reading", "assignment", "quiz"
	Content      any       `json:"content" db:"content"` // Stored as JSONB
	DisplayOrder int       `json:"display_order" db:"display_order"`
	IsCompleted  bool      `json:"is_completed" db:"-"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// --- Structs for the JSONB Content field ---

// VideoContent defines the structure for a "video" content block.
// LastStoppedAt(out): forms the response to be sent back to the frontend.
type VideoContent struct {
	URL           string `json:"url"`
	Duration      int64  `json:"duration_seconds"`
	LastStoppedAt int64  `json:"last_stopped_at" db:"-"`
	Title         string `json:"title,omitempty"`
}

// ReadingContent defines the structure for a "reading" content block.
type ReadingContent struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// AssignmentContent holds a reference to a separate assignments table.
type AssignmentContent struct {
	AssignmentID int64 `json:"assignment_id"`
}

// QuizContent holds a reference to a separate quizzes table.
type QuizContent struct {
	QuizID int64 `json:"quiz_id"`
}

// UserChapterProgress tracks a specific user's progress in a chapter.
type UserChapterProgress struct {
	UserID             string     `json:"user_id" db:"user_id"`
	ChapterID          int64      `json:"chapter_id" db:"chapter_id"`
	ProgressPercentage int        `json:"progress_percentage" db:"progress_percentage"`
	CompletedAt        *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// UserVideoProgress tracks a user's specific progress within a single video content block.
// LastStoppedAt(store): represents the row in user_video_progress database table.
type UserVideoProgress struct {
	UserID         string    `json:"user_id" db:"user_id"`
	ContentBlockID int64     `json:"content_block_id" db:"content_block_id"`
	LastStoppedAt  int64     `json:"last_stopped_at" db:"last_stopped_at"` // in seconds
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// UpdateVideoProgressRequest defines the request body for updating video progress.
// LastStoppedAt(in): represents how the new progress value is transported from the client to the server.
type UpdateVideoProgressRequest struct {
	LastStoppedAt int64 `json:"last_stopped_at" validate:"gte=0"`
}
