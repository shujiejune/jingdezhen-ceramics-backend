package models

// Tag represents a single, reusable tag that can be applied to
// forum posts, portfolio works, or gallery artworks.
type Tag struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}
