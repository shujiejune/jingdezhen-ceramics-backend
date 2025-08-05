package notification

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/pkg/email"
	"log"
)

type ServiceInterface interface {
	CreateNotification(ctx context.Context, trigger models.NotificationTrigger)
	GetUserNotifications(ctx context.Context, userID string, page, limit int) ([]models.Notification, int, error)
	MarkNotificationsAsRead(ctx context.Context, userID string, notificationIDs []int64) error
	GetUnreadNotificationCount(ctx context.Context, userID string) (int, error)
}

type Service struct {
	repo     RepositoryInterface
	emailSvc email.ServiceInterface
}

func NewService(repo RepositoryInterface, emailSvc email.ServiceInterface) ServiceInterface {
	return &Service{repo: repo, emailSvc: emailSvc}
}

// CreateNotification is designed to be called in a goroutine ("fire-and-forget").
func (s *Service) CreateNotification(ctx context.Context, trigger models.NotificationTrigger) {
	// Business Logic: Don't notify a user about their own actions.
	if trigger.RecipientUserID == trigger.ActorUserID {
		return
	}

	// 1. Save the notification to the database.
	notification, err := s.repo.Insert(ctx, trigger)
	if err != nil {
		log.Printf("ERROR: Failed to save notification to DB: %v", err)
		return
	}

	// 2. (Optional) Send a real-time notification via WebSocket if the user is online.
	// webSocketService.Send(trigger.RecipientUserID, notification)

	// 3. (Optional) Send an email notification.
	// You would fetch user preferences to see if they want emails for this type of notification.
	// You would also fetch more details (user names, post titles) to create a rich email.
	// For now, we'll just log it.
	log.Printf("INFO: Created notification %d. In a real app, an email would be sent.", notification.ID)
}

// ... Implementations for GetUserNotifications, MarkNotificationsAsRead, GetUnreadNotificationCount ...
