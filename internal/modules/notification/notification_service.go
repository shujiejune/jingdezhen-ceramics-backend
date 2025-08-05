package notification

import (
	"database/sql"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	// "jingdezhen-ceramics-backend/internal/modules/user"
)

// WebSocketService remains the same.
type WebSocketService interface {
	SendToUser(userID string, notification *models.Notification)
	IsUserOnline(userID string) bool
}

// Service provides business logic for notifications.
type Service interface {
	CreateNotification(params models.CreateNotificationParams) (*models.Notification, error)
	// GetNotificationsForUser should ideally return the composed notifications with ActorUser populated.
	// This composition logic lives here in the service.
	GetNotificationsForUser(userID string, page, pageSize int) ([]models.Notification, int64, error)
	GetUnreadNotificationCount(userID string) (int64, error)
	MarkNotificationAsRead(notificationID int64, userID string) error
	MarkAllNotificationsAsRead(userID string) error
}

type service struct {
	repo Repository
	// userRepo         user.Repository // To fetch user details for messages
	webSocketService WebSocketService
}

// NewService creates a new notification service.
func NewService(repo Repository /* userRepo user.Repository, */, wsService WebSocketService) Service {
	return &service{
		repo: repo,
		// userRepo:         userRepo,
		webSocketService: wsService,
	}
}

// CreateNotification creates a new notification, saves it, and pushes it in real-time if the user is online.
func (s *service) CreateNotification(params models.CreateNotificationParams) (*models.Notification, error) {
	message, err := s.generateMessage(params)
	if err != nil {
		return nil, fmt.Errorf("failed to generate notification message: %w", err)
	}

	notification := &models.Notification{
		RecipientUserID:  params.RecipientUserID,
		NotificationType: params.Type,
		Message:          message,
		IsRead:           false,
	}

	// Handle nullable ActorUserID
	if params.ActorUserID != "" {
		notification.ActorUserID = &params.ActorUserID
	}

	// Handle nullable EntityType and EntityID
	if params.EntityType != "" {
		notification.EntityType = &params.EntityType
	}
	if params.EntityID != 0 {
		notification.EntityID = &params.EntityID
	}

	if err := s.repo.Create(notification); err != nil {
		return nil, err
	}

	// Real-time push logic (remains the same conceptually)
	if s.webSocketService != nil && s.webSocketService.IsUserOnline(params.RecipientUserID) {
		// Before sending, you would populate the ActorUser field here for the real-time message.
		// actor, _ := s.userRepo.FindByID(params.ActorUserID)
		// notification.ActorUser = actor
		s.webSocketService.SendToUser(params.RecipientUserID, notification)
	}

	return notification, nil
}

// GetNotificationsForUser retrieves notifications and composes them with actor details.
func (s *service) GetNotificationsForUser(userID string, page, pageSize int) ([]models.Notification, int64, error) {
	offset := (page - 1) * pageSize

	// 1. Get total count for pagination
	total, err := s.repo.GetTotalCountByUserID(userID)
	if err != nil {
		return nil, 0, err
	}

	// 2. Get the raw notifications from the database
	notifications, err := s.repo.GetByUserID(userID, pageSize, offset)
	if err != nil || len(notifications) == 0 {
		return notifications, total, err
	}

	// 3. **MANUAL ASSOCIATION LOGIC (replaces GORM's Preload)**
	// Collect all unique, non-nil actor IDs
	actorIDs := make(map[string]struct{})
	for _, n := range notifications {
		if n.ActorUserID != nil {
			actorIDs[*n.ActorUserID] = struct{}{}
		}
	}

	// Fetch all required actors in a single DB call
	// actors, err := s.userRepo.FindUsersByIDs(mapKeysToSlice(actorIDs))
	// if err != nil {
	// 	return nil, 0, err // Or handle gracefully
	// }
	// actorsByID := map[string]*models.User{}
	// for _, actor := range actors {
	// 	actorsByID[actor.ID] = actor
	// }

	// 4. Stitch the actor data onto the notifications
	// for i := range notifications {
	// 	if notifications[i].ActorUserID != nil {
	// 		if actor, ok := actorsByID[*notifications[i].ActorUserID]; ok {
	// 			notifications[i].ActorUser = actor
	// 		}
	// 	}
	// }

	return notifications, total, nil
}

func (s *service) GetUnreadNotificationCount(userID string) (int64, error) {
	return s.repo.GetUnreadCount(userID)
}

func (s *service) MarkNotificationAsRead(notificationID int64, userID string) error {
	updated, err := s.repo.MarkAsRead(notificationID, userID)
	if err != nil {
		return err
	}
	if !updated {
		return sql.ErrNoRows // Use a standard error to indicate not found
	}
	return nil
}

func (s *service) MarkAllNotificationsAsRead(userID string) error {
	_, err := s.repo.MarkAllAsRead(userID)
	return err
}

// generateMessage constructs the human-readable notification message.
func (s *service) generateMessage(params models.CreateNotificationParams) (string, error) {
	// NOTE: This function would use the userRepo to get the actor's name.
	// For this example, we'll use the ExtraData map.
	actorName := params.ExtraData["actorName"]
	if actorName == "" {
		actorName = "Someone"
	}

	switch params.Type {
	case models.NotificationTypeCourseUpdate:
		return fmt.Sprintf("The course '%s' has a new update or announcement.", params.ExtraData["courseName"]), nil
	case models.NotificationTypeAssignmentGraded:
		return fmt.Sprintf("Your assignment for '%s' has been graded.", params.ExtraData["chapterName"]), nil
	case models.NotificationTypeForumPostCommented:
		return fmt.Sprintf("%s commented on your forum post: '%s'.", actorName, params.ExtraData["postTitle"]), nil
	case models.NotificationTypeForumCommentReplied:
		return fmt.Sprintf("%s replied to your comment.", actorName), nil
	case models.NotificationTypePortfolioWorkHighlighted:
		return fmt.Sprintf("Congratulations! Your work '%s' has been featured as an Editor's Choice.", params.ExtraData["workTitle"]), nil
	case models.NotificationTypeBadgeEarned:
		return fmt.Sprintf("You've earned the %s badge! Keep up the great work.", params.ExtraData["badgeName"]), nil
	default:
		return "", fmt.Errorf("unrecognized notification type: %s", params.Type)
	}
}
