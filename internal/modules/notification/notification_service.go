package notification

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/internal/modules/user"
	"log"
)

// WebSocketService remains the same.
type WebSocketService interface {
	SendToUser(userID string, notification *models.Notification)
	IsUserOnline(userID string) bool
}

// Service provides business logic for notifications.
type Service interface {
	CreateNotification(ctx context.Context, params models.CreateNotificationParams) (*models.Notification, error)
	// GetNotificationsForUser should ideally return the composed notifications with ActorUser populated.
	// This composition logic lives here in the service.
	GetNotificationsForUser(ctx context.Context, userID string, page, pageSize int) ([]models.Notification, int64, error)
	GetUnreadNotificationCount(ctx context.Context, userID string) (int64, error)
	MarkNotificationAsRead(ctx context.Context, notificationID int64, userID string) error
	MarkAllNotificationsAsRead(ctx context.Context, userID string) error
}

type service struct {
	repo             Repository
	userRepo         user.Repository
	webSocketService WebSocketService
}

// NewService creates a new notification service.
func NewService(repo Repository, userRepo user.Repository, wsService WebSocketService) Service {
	return &service{
		repo:             repo,
		userRepo:         userRepo,
		webSocketService: wsService,
	}
}

// CreateNotification creates a new notification, saves it, and pushes it in real-time if the user is online.
func (s *service) CreateNotification(ctx context.Context, params models.CreateNotificationParams) (*models.Notification, error) {
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

	if err := s.repo.Create(ctx, notification); err != nil {
		return nil, err
	}

	// Real-time push logic
	if s.webSocketService != nil && s.webSocketService.IsUserOnline(params.RecipientUserID) {
		// Before sending, populate the ActorUser field here for the real-time message.
		actor, err := s.userRepo.FindByID(ctx, params.ActorUserID)
		if err != nil {
			log.Printf("WARN: could not find actor user %s for real-time push: %v", params.ActorUserID, err)
		} else {
			notification.ActorUser = actor
		}
		s.webSocketService.SendToUser(params.RecipientUserID, notification)
	}

	return notification, nil
}

// mapKeysToSlice converts the keys of a map[string]struct{} to a slice of strings.
func mapKeysToSlice(m map[string]struct{}) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	return s
}

// GetNotificationsForUser retrieves notifications and composes them with actor details.
func (s *service) GetNotificationsForUser(ctx context.Context, userID string, page, limit int) ([]models.Notification, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// 1. Get total count for pagination
	total, err := s.repo.GetTotalCountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	// 2. Get the raw notifications from the database
	notifications, err := s.repo.GetByUserID(ctx, userID, limit, offset)
	if err != nil || len(notifications) == 0 {
		return notifications, total, err
	}

	// 3. Collect all unique, non-nil actor IDs
	actorIDsSet := make(map[string]struct{})
	for _, n := range notifications {
		if n.ActorUserID != nil {
			actorIDsSet[*n.ActorUserID] = struct{}{}
		}
	}

	// 4. If there are actors to fetch, fetch them all in one query.
	if len(actorIDsSet) > 0 {
		// Convert the set (map keys) to a slice.
		actorIDsSlice := make([]string, 0, len(actorIDsSet))
		for id := range actorIDsSet {
			actorIDsSlice = append(actorIDsSlice, id)
		}

		// Fetch all users in a single database call.
		actors, err := s.userRepo.FindByIDs(ctx, actorIDsSlice)
		if err != nil {
			// Handle gracefully: Log the error but return the notifications anyway.
			log.Printf("ERROR: could not fetch actor users for notifications: %v", err)
		} else {
			// Create a map for quick lookups.
			actorsByID := make(map[string]*models.User, len(actors))
			for i := range actors {
				actorsByID[actors[i].ID] = &actors[i]
			}

			// 5. Stitch the actor data onto the notifications.
			for i := range notifications {
				if notifications[i].ActorUserID != nil {
					if actor, ok := actorsByID[*notifications[i].ActorUserID]; ok {
						notifications[i].ActorUser = actor
					}
				}
			}
		}
	}

	return notifications, total, nil
}

func (s *service) GetUnreadNotificationCount(ctx context.Context, userID string) (int64, error) {
	return s.repo.GetUnreadCount(ctx, userID)
}

func (s *service) MarkNotificationAsRead(ctx context.Context, notificationID int64, userID string) error {
	updated, err := s.repo.MarkAsRead(ctx, notificationID, userID)
	if err != nil {
		return err
	}
	if !updated {
		return models.ErrNotFound
	}
	return nil
}

func (s *service) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	_, err := s.repo.MarkAllAsRead(ctx, userID)
	return err
}

// generateMessage constructs the human-readable notification message.
func (s *service) generateMessage(params models.CreateNotificationParams) (string, error) {
	// NOTE: This function would use the userRepo to get the actor's name.
	actorName := params.ExtraData["actorName"]
	if actorName == "" {
		actorName = "Someone"
	}

	switch params.Type {
	case models.NotificationTypeCourseUpdate:
		return fmt.Sprintf("The course '%s' has a new update or announcement.", params.ExtraData["courseName"]), nil
	case models.NotificationTypeAssignmentGraded:
		return fmt.Sprintf("Your assignment for Course '%s', Chpater '%s' has been graded.", params.ExtraData["courseName"], params.ExtraData["chapterName"]), nil
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
