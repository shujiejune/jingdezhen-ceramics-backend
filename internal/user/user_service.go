package user

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/forum" // For publishing notes
	"jingdezhen-ceramics-backend/internal/models"
	// "golang.org/x/crypto/bcrypt" // If handling password hashing here
	"log" // For contact form simulation
)

// ServiceInterface defines methods for user business logic.
type ServiceInterface interface {
	GetUserProfile(ctx context.Context, userID string) (*models.User, error)
	UpdateUserProfile(ctx context.Context, userID string, data models.UserUpdateData) (*models.User, error)
	HandleContactSubmission(ctx context.Context, data models.ContactFormData) error

	// User Notes
	ListUserNotes(ctx context.Context, userID string, page, limit int) ([]models.UserNote, int, error)
	GetUserNoteDetails(ctx context.Context, userID string, noteID int) (*models.UserNote, error)
	CreateUserNote(ctx context.Context, userID string, data models.CreateUserNoteData) (*models.UserNote, error)
	UpdateUserNote(ctx context.Context, userID string, noteID int, data models.UpdateUserNoteData) (*models.UserNote, error)
	DeleteUserNote(ctx context.Context, userID string, noteID int) error
	PublishNoteToForum(ctx context.Context, userID string, noteID int, publishDetails models.ForumPostPublishDetails) (*models.ForumPost, error)

	// Admin
	AdminListUsers(ctx context.Context, page, limit int) ([]models.User, int, error)
	AdminUpdateUserRole(ctx context.Context, targetUserID string, newRole string) error
}

type Service struct {
	userRepo RepositoryInterface
	// For simplicity, userNote specific methods are on RepositoryInterface for now.
	// In a larger system, userNoteRepo might be a separate RepositoryInterface.
	forumSvc forum.ServiceInterface // Injected for publishing notes
	// emailSvc     email.ServiceInterface // For sending contact emails
}

func NewService(
	userRepo RepositoryInterface,
	forumSvc forum.ServiceInterface,
	// emailSvc email.ServiceInterface,
) ServiceInterface {
	return &Service{
		userRepo: userRepo,
		forumSvc: forumSvc,
		// emailSvc: emailSvc,
	}
}

func (s *Service) GetUserProfile(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		// Map repository errors to service-level errors if needed
		return nil, fmt.Errorf("service.GetUserProfile: %w", err)
	}
	return user, nil
}

func (s *Service) UpdateUserProfile(ctx context.Context, userID string, data models.UserUpdateData) (*models.User, error) {
	// Add validation logic here if not handled by handler/framework
	// e.g., check if nickname is unique if that's a requirement (would need repo method)
	updatedUser, err := s.userRepo.Update(ctx, userID, data)
	if err != nil {
		return nil, fmt.Errorf("service.UpdateUserProfile: %w", err)
	}
	return updatedUser, nil
}

func (s *Service) HandleContactSubmission(ctx context.Context, data models.ContactFormData) error {
	// In a real application, this would:
	// 1. Validate the data (already done by handler with 'validate' tags usually)
	// 2. Sanitize inputs
	// 3. Send an email to the admin using an email service
	// 4. Optionally, save the contact message to the database
	log.Printf("Contact Form Submitted: Name: %s, Email: %s, Subject: %s, Message: %s",
		data.Name, data.Email, data.Subject, data.Message)

	// Example: err := s.emailSvc.SendContactEmailToAdmin(ctx, data)
	// if err != nil {
	//     return fmt.Errorf("service.HandleContactSubmission: failed to send email: %w", err)
	// }
	return nil // Simulate success
}

// --- User Notes ---
func (s *Service) ListUserNotes(ctx context.Context, userID string, page, limit int) ([]models.UserNote, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	} // Default/max limit
	notes, total, err := s.userRepo.ListUserNotes(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListUserNotes: %w", err)
	}
	return notes, total, nil
}

func (s *Service) GetUserNoteDetails(ctx context.Context, userID string, noteID int) (*models.UserNote, error) {
	note, err := s.userRepo.GetUserNoteByID(ctx, noteID, userID) // Repo checks ownership
	if err != nil {
		return nil, fmt.Errorf("service.GetUserNoteDetails: %w", err)
	}
	return note, nil
}

func (s *Service) CreateUserNote(ctx context.Context, userID string, data models.CreateUserNoteData) (*models.UserNote, error) {
	// Add business logic: e.g., check if user can create notes for this entity_type/entity_id
	note, err := s.userRepo.CreateUserNote(ctx, userID, data)
	if err != nil {
		return nil, fmt.Errorf("service.CreateUserNote: %w", err)
	}
	return note, nil
}

func (s *Service) UpdateUserNote(ctx context.Context, userID string, noteID int, data models.UpdateUserNoteData) (*models.UserNote, error) {
	// userRepo.UpdateUserNote already checks ownership by including userID in query
	note, err := s.userRepo.UpdateUserNote(ctx, noteID, userID, data)
	if err != nil {
		return nil, fmt.Errorf("service.UpdateUserNote: %w", err)
	}
	return note, nil
}

func (s *Service) DeleteUserNote(ctx context.Context, userID string, noteID int) error {
	// userRepo.DeleteUserNote already checks ownership by including userID in query
	err := s.userRepo.DeleteUserNote(ctx, noteID, userID)
	if err != nil {
		return fmt.Errorf("service.DeleteUserNote: %w", err)
	}
	return nil
}

func (s *Service) PublishNoteToForum(ctx context.Context, userID string, noteID int, publishDetails models.ForumPostPublishDetails) (*models.ForumPost, error) {
	note, err := s.userRepo.GetUserNoteByID(ctx, noteID, userID)
	if err != nil {
		return nil, fmt.Errorf("service.PublishNoteToForum.GetNote: %w", err)
	}
	if note.IsPublishedToForum {
		// Optionally, you could return the existing forum post if note.ForumPostID is not nil
		return nil, models.ErrConflict // Define this error: "note already published"
	}

	// Prepare data for creating forum post
	createPostData := models.CreateForumPostData{ // Assuming this struct exists in models
		Title:      publishDetails.Title,
		Content:    note.Content, // Use content from the note
		CategoryID: publishDetails.CategoryID,
		Tags:       publishDetails.Tags,
		// UserID is handled by forumService.CreatePost based on the authenticated user
	}

	createdPost, err := s.forumSvc.CreatePost(ctx, userID, createPostData) // userID passed here is the authenticated user
	if err != nil {
		return nil, fmt.Errorf("service.PublishNoteToForum.CreatePost: %w", err)
	}

	// Mark note as published
	err = s.userRepo.MarkNoteAsPublished(ctx, noteID, createdPost.ID)
	if err != nil {
		// Log this error but don't fail the whole operation as post is created
		log.Printf("ERROR: service.PublishNoteToForum.MarkNoteAsPublished for noteID %d, postID %d: %v", noteID, createdPost.ID, err)
	}
	return createdPost, nil
}

// --- Admin Service Methods ---
func (s *Service) AdminListUsers(ctx context.Context, page, limit int) ([]models.User, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.userRepo.ListAll(ctx, page, limit)
}

func (s *Service) AdminUpdateUserRole(ctx context.Context, targetUserID string, newRole string) error {
	// Add validation for newRole if it's not a predefined valid role
	if newRole != models.RoleAdmin && newRole != models.RoleNormalUser {
		return fmt.Errorf("service.AdminUpdateUserRole: invalid role '%s'", newRole)
	}
	// Check if targetUserID exists
	_, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("service.AdminUpdateUserRole: target user not found: %w", err)
	}

	return s.userRepo.UpdateRole(ctx, targetUserID, newRole)
}
