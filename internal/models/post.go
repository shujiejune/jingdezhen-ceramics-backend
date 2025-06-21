package models

import "time"

type ForumPost struct {
	ID             int64     `json:"forum_post_id" db:"forum_post_id"`
	UserID         string    `json:"user_id" db:"user_id"`
	AuthorNickname string    `json:"author_nickname" db:"author_nickname"`
	Title          string    `json:"title" db:"title"`
	Content        string    `json:"content" db:"content"`
	CategoryID     int       `json:"category_id" db:"category_id"`
	CategoryName   string    `json:"category_name" db:"category_name"`
	Tags           []string  `json:"tags" db:"tags"`
	IsPinned       bool      `json:"is_pinned" db:"is_pinned"`
	ViewCount      int       `json:"view_count" db:"view_count"`
	CommentCount   int       `json:"comment_count" db:"comment_count"`
	LikeCount      int       `json:"like_count" db:"like_count"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	LastActivityAt time.Time `json:"last_activity_at" db:"last_activity_at"`
}

type CreateForumPostData struct {
	Title      string   `json:"title" validate:"required,min=3,max=255"`
	Content    string   `json:"content" validate:"required,min=10"`
	CategoryID int      `json:"category_id" validate:"required,gt=0"`
	Tags       []string `json:"tags,omitempty"` // Or []int for tag IDs
}

type UserSavedPostEntry struct {
	Post    ForumPost `json:"post"`
	SavedAt time.Time `json:"saved_at"`
}
