package notification

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
	"jingdezhen-ceramics-backend/internal/models"
)

// Repository provides database access for notifications using raw SQL.
type Repository interface {
	Create(notification *models.Notification) error
	GetByUserID(userID string, limit, offset int) ([]models.Notification, error)
	GetTotalCountByUserID(userID string) (int64, error)
	MarkAsRead(notificationID int64, userID string) (bool, error)
	MarkAllAsRead(userID string) (int64, error)
	GetUnreadCount(userID string) (int64, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates a new notification repository.
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db}
}

// Create saves a new notification to the database and populates the generated ID and CreatedAt fields.
func (r *repository) Create(notification *models.Notification) error {
	query := `
        INSERT INTO notifications (recipient_user_id, actor_user_id, notification_type, entity_type, entity_id, message, is_read)
        VALUES (:recipient_user_id, :actor_user_id, :notification_type, :entity_type, :entity_id, :message, :is_read)
        RETURNING id, created_at`

	// Use NamedQuery to map struct fields to the query placeholders.
	rows, err := r.db.NamedQuery(query, notification)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Scan the returned id and created_at back into the notification struct.
	if rows.Next() {
		return rows.StructScan(notification)
	}
	return rows.Err()
}

// GetByUserID retrieves a paginated list of notifications for a specific user.
func (r *repository) GetByUserID(userID string, limit, offset int) ([]models.Notification, error) {
	var notifications []models.Notification
	query := `
        SELECT id, recipient_user_id, actor_user_id, notification_type, entity_type, entity_id, message, is_read, created_at
        FROM notifications
        WHERE recipient_user_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3`
	err := r.db.Select(&notifications, query, userID, limit, offset)
	return notifications, err
}

// GetTotalCountByUserID gets the total number of notifications for a user (for pagination).
func (r *repository) GetTotalCountByUserID(userID string) (int64, error) {
	var total int64
	query := `SELECT COUNT(*) FROM notifications WHERE recipient_user_id = $1`
	err := r.db.Get(&total, query, userID)
	return total, err
}

// MarkAsRead marks a single notification as read. Returns true if a row was updated.
func (r *repository) MarkAsRead(notificationID int64, userID string) (bool, error) {
	query := `UPDATE notifications SET is_read = TRUE WHERE id = $1 AND recipient_user_id = $2`
	result, err := r.db.Exec(query, notificationID, userID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected > 0, err
}

// MarkAllAsRead marks all notifications for a user as read. Returns the number of rows updated.
func (r *repository) MarkAllAsRead(userID string) (int64, error) {
	query := `UPDATE notifications SET is_read = TRUE WHERE recipient_user_id = $1 AND is_read = FALSE`
	result, err := r.db.Exec(query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetUnreadCount retrieves the count of unread notifications for a user.
func (r *repository) GetUnreadCount(userID string) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM notifications WHERE recipient_user_id = $1 AND is_read = FALSE`
	err := r.db.Get(&count, query, userID)
	return count, err
}

// convertStringToNullString handles converting an empty string ActorUserID to a null value for the database.
func convertStringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
