package notification

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	Insert(ctx context.Context, trigger models.NotificationTrigger) (*models.Notification, error)
	FindForUser(ctx context.Context, userID string, page, limit int) ([]models.Notification, int, error)
	MarkAsRead(ctx context.Context, userID string, notificationIDs []int64) error
	GetUnreadCount(ctx context.Context, userID string) (int, error)
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) RepositoryInterface {
	return &Repository{db: db}
}

func (r *Repository) Insert(ctx context.Context, trigger models.NotificationTrigger) (*models.Notification, error) {
	var notification models.Notification
	query := `
		INSERT INTO notifications (recipient_user_id, actor_user_id, action_type, entity_type, entity_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, recipient_user_id, actor_user_id, action_type, entity_type, entity_id, is_read, created_at
	`
	err := r.db.QueryRow(ctx, query,
		trigger.RecipientUserID, trigger.ActorUserID, trigger.ActionType, trigger.EntityType, trigger.EntityID,
	).Scan(
		&notification.ID, &notification.RecipientUserID, &notification.ActorUserID, &notification.ActionType,
		&notification.EntityType, &notification.EntityID, &notification.IsRead, &notification.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.InsertNotification: %w", err)
	}
	return &notification, nil
}

// ... Implementations for FindForUser, MarkAsRead, GetUnreadCount ...
