package models

import "time"

// PortfolioWorkImage represents a single image for a portfolio work.
type PortfolioWorkImage struct {
	ID          int64  `json:"id" db:"id"`
	ImageURL    string `json:"image_url" db:"image_url"`
	IsThumbnail bool   `json:"is_thumbnail" db:"is_thumbnail"`
	Caption     string `json:"caption,omitempty" db:"caption"`
}

// PortfolioWork represents a single creative work submitted by a student.
type PortfolioWork struct {
	ID              int64                `json:"id" db:"id"`
	UserID          string               `json:"user_id" db:"user_id"`
	CreatorNickname string               `json:"creator_nickname" db:"creator_nickname"`
	Title           string               `json:"title" db:"title"`
	Description     string               `json:"description,omitempty" db:"description"` // Full description for detail view
	IsEditorsChoice bool                 `json:"is_editors_choice" db:"is_editors_choice"`
	KudosCount      int                  `json:"kudos_count" db:"kudos_count"`
	CreatedAt       time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at" db:"updated_at"`
	ThumbnailURL    string               `json:"thumbnail_url,omitempty" db:"-"` // Derived from images for list view
	Images          []PortfolioWorkImage `json:"images,omitempty" db:"-"`        // For detail view
	Tags            []string             `json:"tags,omitempty" db:"-"`          // Loaded separately
	// User-specific field, populated in the service layer
	HasMyKudo bool `json:"has_my_kudo" db:"-"`
}

// PortfolioFilters defines the available query parameters for filtering portfolio works.
type PortfolioFilters struct {
	Page     int
	Limit    int
	Category string // This will filter by tags
	Sort     string // "kudos", "latest" (default)
}

// CreateWorkRequest defines the request body for creating a new portfolio work.
type CreateWorkRequest struct {
	Title             string   `json:"title" validate:"required,min=3,max=255"`
	Description       string   `json:"description" validate:"required,min=10"`
	ImageURLs         []string `json:"image_urls" validate:"required,min=1,dive,url"`
	ThumbnailURLIndex *int     `json:"thumbnail_url_index,omitempty" validate:"omitempty,gte=0"` // Index in ImageURLs to use as thumbnail
	Tags              []string `json:"tags,omitempty"`
}

// UpdateWorkRequest defines the request body for updating a portfolio work.
type UpdateWorkRequest struct {
	Title       *string  `json:"title,omitempty" validate:"omitempty,min=3,max=255"`
	Description *string  `json:"description,omitempty" validate:"omitempty,min=10"`
	Tags        []string `json:"tags,omitempty"` // For now, images are not updatable via this endpoint
}
