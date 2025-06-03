package models

import "time"

// UserNote represents a user's private note
type UserNote struct {
	ID                 int       `json:"id" db:"id"`
	UserID             string    `json:"user_id" db:"user_id"`
	Title              string    `json:"title" db:"title"`
	Content            string    `json:"content" db:"content"`
	EntityType         string    `json:"entity_type" db:"entity_type"` // e.g., "artwork", "course_chapter"
	EntityID           int       `json:"entity_id" db:"entity_id"`
	IsPublishedToForum bool      `json:"is_published_to_forum" db:"is_published_to_forum"`
	ForumPostID        *int      `json:"forum_post_id,omitempty" db:"forum_post_id"` // Pointer to allow NULL
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// CreateUserNoteData is the data needed to create a new user note
type CreateUserNoteData struct {
	Title      string `json:"title" validate:"required,max=255"`
	Content    string `json:"content" validate:"required"`
	EntityType string `json:"entity_type" validate:"required,oneof=artwork course_chapter"`
	EntityID   int    `json:"entity_id" validate:"required,gt=0"`
}

// UpdateUserNoteData is the data needed to update a user note
type UpdateUserNoteData struct {
	Title   *string `json:"title,omitempty" validate:"omitempty,max=255"`
	Content *string `json:"content,omitempty"`
}

// ForumPostPublishDetails holds details for publishing a note to the forum
type ForumPostPublishDetails struct {
	Title      string   `json:"title" validate:"required,max=255"`
	CategoryID int      `json:"category_id" validate:"required,gt=0"`
	Tags       []string `json:"tags,omitempty"` // Or []int for tag IDs
}
