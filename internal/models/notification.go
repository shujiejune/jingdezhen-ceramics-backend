package models

import "time"

// NotificationType defines the type of notification.
type NotificationType string

const (
	// Course-related notifications
	NotificationTypeCourseUpdate     NotificationType = "course_update"
	NotificationTypeAssignmentGraded NotificationType = "assignment_graded"

	// Forum-related notifications
	NotificationTypeForumPostLiked      NotificationType = "forum_post_liked"
	NotificationTypeForumPostCommented  NotificationType = "forum_post_commented"
	NotificationTypeForumCommentLiked   NotificationType = "forum_comment_liked"
	NotificationTypeForumCommentReplied NotificationType = "forum_comment_replied"

	// Portfolio-related notifications
	NotificationTypePortfolioWorkHighlighted NotificationType = "portfolio_work_highlighted"

	// Badge-related notifications
	NotificationTypeBadgeEarned NotificationType = "badge_earned"
)

// Notification represents a single notification for a user.
type Notification struct {
	ID               int64            `json:"notification_id" db:"notification_id"`
	RecipientUserID  string           `json:"recipient_user_id" db:"recipient_user_id"`
	ActorUserID      *string          `json:"actor_user_id,omitempty" db:"actor_user_id"`
	ActorUser        *User            `json:"actor_user" db:"-"`
	NotificationType NotificationType `json:"notification_type" db:"notification_type"`
	EntityType       string           `json:"entity_type,omitempty" db:"entity_type"`
	EntityID         *int64           `json:"entity_id,omitempty" db:"entity_id"`
	Message          string           `json:"message" db:"message"`
	IsRead           bool             `json:"is_read" db:"is_read"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
}

// CreateNotificationParams holds the data needed to create a notification.
type CreateNotificationParams struct {
	RecipientUserID string
	ActorUserID     string // Use an empty string for system notifications (e.g., badges)
	Type            NotificationType
	EntityType      string
	EntityID        int64
	ExtraData       map[string]string
}
