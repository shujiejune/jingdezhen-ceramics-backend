package models

import "time"

type Notification struct {
	ID              string    `json:"notification_id" db:"notification_id"`
	RecipientUserID string    `json:"recipient_user_id" db:"recipient_user_id"`
	ActorUserID     string    `json:"actor_user_id" db:"actor_user_id"`
	ActionType      string    `json:"action_type" db:"action_type"` // e.g. "kudo_portfolio_work", "comment_forum_post"
	EntityType      string    `json:"entity_type" db:"entity_type"` // e.g. "portfolio_work", "forum_post"
	EntityID        int       `json:"entity_id" db:"entity_id"`
	Message         string    `json:"message" db:"message"`
	IsRead          bool      `json:"is_read" db:"is_read"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}
