package models

import "time"

// PortfolioWorkImage represents a single image for a portfolio work.
type PortfolioWorkImage struct {
	ID           int64  `json:"id" db:"id"`
	ImageURL     string `json:"image_url" db:"image_url"`
	IsThumbnail  bool   `json:"is_thumbnail" db:"is_thumbnail"`
	Caption      string `json:"caption,omitempty" db:"caption"`
	DisplayOrder int    `json:"display_order" db:"display_order"`
}

// PortfolioWork represents a single creative work submitted by a student.
type PortfolioWork struct {
	ID              int64                `json:"id" db:"id"`
	UserID          string               `json:"user_id" db:"user_id"`
	CreatorNickname string               `json:"creator_nickname" db:"creator_nickname"`
	Title           string               `json:"title" db:"title"`
	Description     *string              `json:"description,omitempty" db:"description"`
	IsEditorsChoice bool                 `json:"is_editors_choice" db:"is_editors_choice"`
	UpvotesCount    int                  `json:"upvotes_count" db:"upvotes_count"`
	CreatedAt       time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at" db:"updated_at"`
	ThumbnailURL    string               `json:"thumbnail_url,omitempty" db:"-"` // Derived from images for list view
	Images          []PortfolioWorkImage `json:"images,omitempty" db:"-"`        // For detail view
	Tags            []Tag                `json:"tags,omitempty" db:"-"`          // Loaded separately
	UpvotedByMe     bool                 `json:"upvoted_by_me" db:"-"`           //user-specific
	SavedByMe       bool                 `json:"saved_by_me" db:"-"`             //user-specific
}

// PortfolioFilters defines the available query parameters for filtering portfolio works.
type PortfolioFilters struct {
	Page  int
	Limit int
	Sort  string // "upvotes", "latest" (default)
	Tags  []Tag
}

// CreateWorkRequest defines the request body for creating a new portfolio work.
type CreateWorkRequest struct {
	Title             string   `json:"title" validate:"required,min=3,max=255"`
	Description       *string  `json:"description" validate:"required,min=10"`
	ImageURLs         []string `json:"image_urls" validate:"required,min=1,max=8,dive,url"`
	ThumbnailURLIndex *int     `json:"thumbnail_url_index,omitempty" validate:"omitempty,gte=0"` // Index in ImageURLs to use as thumbnail
	Tags              []Tag    `json:"tags,omitempty"`
}

// UpdateWorkRequest defines the request body for updating a portfolio work.
type UpdateWorkRequest struct {
	Title       *string              `json:"title,omitempty" validate:"omitempty,min=3,max=255"`
	Description *string              `json:"description,omitempty" validate:"omitempty,min=10"`
	Images      []PortfolioWorkImage `json:"images,omitempty" validate:"max=8"`
	Tags        []Tag                `json:"tags,omitempty"`
}

// ToggleUpvoteResult is a generic response for upvote/downvote actions.
type ToggleUpvoteResult struct {
	IsUpvoted bool `json:"is_upvoted"`
	NewCount  int  `json:"new_count"`
}
