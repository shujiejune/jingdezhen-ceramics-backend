package models

// Badge defines the properties of an achievable badge.
type Badge struct {
	ID          int            `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`
	IconURL     string         `json:"icon_url" db:"icon_url"`
	Criteria    map[string]any `json:"-" db:"criteria"`
}
