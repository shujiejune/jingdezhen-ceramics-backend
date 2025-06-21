package models

import "time"

// UserNote represents a user's private note
type UserNote struct {
	ID                 int            `json:"id" db:"id"`
	UserID             string         `json:"user_id" db:"user_id"`
	Title              string         `json:"title" db:"title"`
	Content            string         `json:"content" db:"content"`
	EntityType         *string        `json:"entity_type,omitempty" db:"entity_type"` // e.g., "artwork", "course_chapter"
	EntityID           *int           `json:"entity_id,omitempty" db:"entity_id"`
	IsPublishedToForum bool           `json:"is_published_to_forum" db:"is_published_to_forum"`
	ForumPostID        *int           `json:"forum_post_id,omitempty" db:"forum_post_id"` // Pointer to allow NULL
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at" db:"updated_at"`
	Links              []UserNoteLink `json:"links,omitempty" db:"-"`
}

// UserNoteLink represents a link from a user note to another entity
type UserNoteLink struct {
	ID                   int       `json:"id" db:"id"`
	UserNoteID           int       `json:"user_note_id" db:"user_note_id"`
	LinkedEntityType     string    `json:"linked_entity_type" db:"linked_entity_type"`
	LinkedEntityIDInt    *int      `json:"linked_entity_id_int,omitempty" db:"linked_entity_id_int"`
	LinkedEntityIDUUID   *string   `json:"linked_entity_id_uuid,omitempty" db:"linked_entity_id_uuid"` // Assuming UUIDs are strings
	LinkedEntityIDString *string   `json:"linked_entity_id_string,omitempty" db:"linked_entity_id_string"`
	LinkDescription      string    `json:"link_description,omitempty" db:"link_description"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
}

// CreateUserNoteData is the data needed to create a new user note
type CreateUserNoteData struct {
	Title   string `json:"title" validate:"required,max=255"`
	Content string `json:"content" validate:"required"`
	// Optional: Initial primary association
	EntityType *string `json:"entity_type,omitempty" validate:"omitempty,oneof=artwork course_chapter"`
	EntityID   *int    `json:"entity_id,omitempty" validate:"omitempty,gt=0"`
}

// UpdateUserNoteData is the data needed to update a user note
type UpdateUserNoteData struct {
	Title   *string `json:"title,omitempty" validate:"omitempty,max=255"`
	Content *string `json:"content,omitempty"`
}

// Data to add a link to a note
type AddLinkToNoteData struct {
	LinkedEntityType     string  `json:"linked_entity_type" validate:"required"`
	LinkedEntityIDInt    *int    `json:"linked_entity_id_int,omitempty" validate:"omitempty,gt=0"`
	LinkedEntityIDUUID   *string `json:"linked_entity_id_uuid,omitempty" validate:"omitempty,uuid"`
	LinkedEntityIDString *string `json:"linked_entity_id_string,omitempty"`
	LinkDescription      string  `json:"link_description,omitempty"`
}

// ForumPostPublishDetails holds details for publishing a note to the forum
type ForumPostPublishDetails struct {
	Title      string   `json:"title" validate:"required,max=255"`
	CategoryID int      `json:"category_id" validate:"required,gt=0"`
	Tags       []string `json:"tags,omitempty"` // Or []int for tag IDs
}
