package models

import "time"

// ForumCategory represents a category for forum posts.
type ForumCategory struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	Description  string `json:"description,omitempty" db:"description"`
	DisplayOrder int    `json:"desplay_order" db:"display_order"`
}

// Tag represents a tag that can be applied to posts.
type Tag struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
	// Could add a 'PostCount' for tag cloud functionality
	PostCount int `json:"post_count,omitempty" db:"-"`
}

// ForumComment represents a single comment in a thread.
type ForumComment struct {
	ID              int64     `json:"id" db:"id"`
	PostID          int64     `json:"post_id" db:"post_id"`
	UserID          string    `json:"user_id" db:"user_id"`
	AuthorNickname  string    `json:"author_nickname" db:"author_nickname"`
	AuthorAvatarURL *string   `json:"author_avatar_url,omitempty" db:"author_avatar_url"`
	ParentCommentID *int64    `json:"parent_comment_id,omitempty" db:"parent_comment_id"`
	Content         string    `json:"content" db:"content"`
	LikeCount       int       `json:"like_count" db:"like_count"`
	IsLikedByMe     bool      `json:"is_liked_by_me" db:"-"` // User-specific
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	// This field is for the frontend to build the nested structure.
	Replies []*ForumComment `json:"replies,omitempty" db:"-"`
}

// ForumPost represents a single forum post (thread).
type ForumPost struct {
	ID              int64     `json:"id" db:"id"`
	UserID          string    `json:"user_id" db:"user_id"`
	AuthorNickname  string    `json:"author_nickname" db:"author_nickname"`
	AuthorAvatarURL *string   `json:"author_avatar_url,omitempty" db:"author_avatar_url"`
	CategoryID      int64     `json:"category_id" db:"category_id"`
	CategoryName    string    `json:"category_name" db:"category_name"`
	Title           string    `json:"title" db:"title"`
	Content         string    `json:"content,omitempty" db:"content"` // Omitted in list view
	IsPinned        bool      `json:"is_pinned" db:"is_pinned"`
	ViewCount       int       `json:"view_count" db:"view_count"`
	CommentCount    int       `json:"comment_count" db:"comment_count"`
	LikeCount       int       `json:"like_count" db:"like_count"`
	LastActivityAt  time.Time `json:"last_activity_at" db:"last_activity_at"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	Tags            []string  `json:"tags,omitempty" db:"-"` // Loaded separately
	// User-specific fields, populated in the service layer
	IsLikedByMe bool `json:"is_liked_by_me" db:"-"`
	IsSavedByMe bool `json:"is_saved_by_me" db:"-"`
}

// PostFilters defines query parameters for filtering posts.
type PostFilters struct {
	Page       int
	Limit      int
	Sort       string // "latest", "hottest"
	Tag        string
	CategoryID int64
}

// CreatePostRequest defines the body for creating a new post.
type CreatePostRequest struct {
	Title      string   `json:"title" validate:"required,min=3,max=255"`
	Content    string   `json:"content" validate:"required,min=10"`
	CategoryID int64    `json:"category_id" validate:"required,gt=0"`
	Tags       []string `json:"tags,omitempty"`
}

// UpdatePostRequest defines the body for updating a post.
type UpdatePostRequest struct {
	Title      *string  `json:"title,omitempty" validate:"omitempty,min=3,max=255"`
	Content    *string  `json:"content,omitempty" validate:"omitempty,min=10"`
	CategoryID *int64   `json:"category_id,omitempty" validate:"omitempty,gt=0"`
	Tags       []string `json:"tags,omitempty"`
}

// CreateCommentRequest defines the body for creating a comment.
type CreateCommentRequest struct {
	Content string `json:"content" validate:"required,min=1"`
}
