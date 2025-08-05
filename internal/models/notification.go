package models

import "time"

// Notification represents a single notification for a user.
type Notification struct {
	ID              string    `json:"notification_id" db:"notification_id"`
	RecipientUserID string    `json:"recipient_user_id" db:"recipient_user_id"`
	ActorUserID     *string   `json:"actor_user_id,omitempty" db:"actor_user_id"`
	ActorNickname   *string   `json:"actor_nickname,omitempty" db:"-"`
	ActionType      string    `json:"action_type" db:"action_type"`           // e.g. "comment_forum_post"
	EntityType      string    `json:"entity_type,omitempty" db:"entity_type"` // e.g. "portfolio_work", "forum_post"
	EntityID        *int      `json:"entity_id,omitempty" db:"entity_id"`
	EntityTitle     *string   `json:"entity_title,omitempty" db:"-"`
	IsRead          bool      `json:"is_read" db:"is_read"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// NotificationTrigger holds the data needed to create a notification.
type NotificationTrigger struct {
	RecipientUserID string
	ActorUserID     string
	ActionType      string
	EntityType      string
	EntityID        int64
	EntityOwnerID   string // The ID of the user who owns the entity (e.g., the post author)
}
