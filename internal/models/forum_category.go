package models

type ForumCategory struct {
	ID           int    `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	Description  string `json:"description" db:"description"`
	DisplayOrder int    `json:"desplay_order" db:"display_order"`
}
