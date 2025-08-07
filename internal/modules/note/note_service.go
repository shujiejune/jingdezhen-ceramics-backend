package note

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/internal/modules/forum"
	"log"
)

type ServiceInterface interface {
	ListUserNotes(ctx context.Context, userID string, page, limit int) ([]models.UserNote, int, error)
	GetUserNoteDetails(ctx context.Context, userID string, noteID int64) (*models.UserNote, error)
	CreateUserNote(ctx context.Context, userID string, data models.CreateUserNoteData) (*models.UserNote, error)
	UpdateUserNote(ctx context.Context, userID string, noteID int64, data models.UpdateUserNoteData) (*models.UserNote, error)
	DeleteUserNote(ctx context.Context, userID string, noteID int64) error
	AddLinkToNote(ctx context.Context, noteID int64, data models.AddLinkToNoteData) (*models.UserNoteLink, error)
	RemoveLinkFromNote(ctx context.Context, noteID, linkID int64) error
	PublishNoteToForum(ctx context.Context, userID string, noteID int64, publishDetails models.ForumPostPublishDetails) (*models.ForumPost, error)
}

type Service struct {
	repo     RepositoryInterface
	forumSvc forum.ServiceInterface
}

func NewService(
	repo RepositoryInterface,
	forumSvc forum.ServiceInterface,
) ServiceInterface {
	return &Service{
		repo:     repo,
		forumSvc: forumSvc,
	}
}

func (s *Service) ListUserNotes(ctx context.Context, userID string, page, limit int) ([]models.UserNote, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	} // Default/max limit
	notes, total, err := s.repo.ListUserNotes(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListUserNotes: %w", err)
	}
	return notes, total, nil
}

func (s *Service) GetUserNoteDetails(ctx context.Context, userID string, noteID int64) (*models.UserNote, error) {
	note, err := s.repo.GetUserNoteByID(ctx, noteID, userID) // Repo checks ownership
	if err != nil {
		return nil, fmt.Errorf("service.GetUserNoteDetails: %w", err)
	}
	links, err := s.repo.GetLinksForNote(ctx, noteID)
	if err != nil {
		log.Printf("Failed to get links for note %d", noteID)
		return note, models.ErrNotFound
	}
	note.Links = links
	return note, nil
}

func (s *Service) CreateUserNote(ctx context.Context, userID string, data models.CreateUserNoteData) (*models.UserNote, error) {
	// Add business logic: e.g., check if user can create notes for this entity_type/entity_id
	note, err := s.repo.CreateUserNote(ctx, userID, data)
	if err != nil {
		return nil, fmt.Errorf("service.CreateUserNote: %w", err)
	}
	return note, nil
}

func (s *Service) UpdateUserNote(ctx context.Context, userID string, noteID int64, data models.UpdateUserNoteData) (*models.UserNote, error) {
	// repo.UpdateUserNote already checks ownership by including userID in query
	note, err := s.repo.UpdateUserNote(ctx, noteID, userID, data)
	if err != nil {
		return nil, fmt.Errorf("service.UpdateUserNote: %w", err)
	}
	return note, nil
}

func (s *Service) DeleteUserNote(ctx context.Context, userID string, noteID int64) error {
	// repo.DeleteUserNote already checks ownership by including userID in query
	err := s.repo.DeleteUserNote(ctx, noteID, userID)
	if err != nil {
		return fmt.Errorf("service.DeleteUserNote: %w", err)
	}
	return nil
}

func (s *Service) AddLinkToNote(ctx context.Context, noteID int64, data models.AddLinkToNoteData) (*models.UserNoteLink, error) {
	link, err := s.repo.AddLinkToNote(ctx, noteID, data)
	if err != nil {
		return nil, fmt.Errorf("service.AddLinkToNote: %w", err)
	}
	return link, nil
}

func (s *Service) RemoveLinkFromNote(ctx context.Context, noteID, linkID int64) error {
	err := s.repo.RemoveLinkFromNote(ctx, noteID, linkID)
	if err != nil {
		return fmt.Errorf("service.RemoveLinkFromNote: %w", err)
	}
	return nil
}

func (s *Service) PublishNoteToForum(ctx context.Context, userID string, noteID int64, publishDetails models.ForumPostPublishDetails) (*models.ForumPost, error) {
	// 1. Validate category ID before starting the transaction.
	isValidCategory, err := s.forumSvc.IsValidCategory(ctx, publishDetails.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate category: %w", err)
	}
	if !isValidCategory {
		return nil, models.ErrInvalidForumPostCategoryID
	}

	// 2. Start a transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("service.PublishNoteToForum.BeginTx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 3. Create transaction-scoped repositories
	noteRepoWithTx := s.repo.WithTx(tx)
	// You don't need a transactional forum repo here, as the forum service will manage its own transaction if needed,
	// but for complex cross-module transactions, you would pass the `tx` object to the other service.
	// For simplicity here, we'll let the forum service handle its own logic.

	// 4. Perform operations
	note, err := noteRepoWithTx.GetUserNoteByID(ctx, noteID, userID)
	if err != nil {
		return nil, fmt.Errorf("service.PublishNoteToForum.GetNote: %w", err)
	}
	if note.IsPublishedToForum {
		return nil, models.ErrConflict // Note is already published
	}

	// Prepare data for creating the forum post
	createPostData := models.CreatePostRequest{
		Title:      publishDetails.Title,
		Content:    note.Content, // Use content from the note
		CategoryID: publishDetails.CategoryID,
		Tags:       publishDetails.Tags,
	}

	// Call the forum service to create the post.
	// The forum service's CreatePost method will handle its own repository interactions.
	createdPost, err := s.forumSvc.CreatePost(ctx, userID, createPostData)
	if err != nil {
		return nil, fmt.Errorf("service.PublishNoteToForum.CreatePost: %w", err)
	}

	// Mark the note as published using the transactional note repository
	err = noteRepoWithTx.MarkNoteAsPublished(ctx, noteID, createdPost.ID)
	if err != nil {
		return nil, fmt.Errorf("service.PublishNoteToForum.MarkNoteAsPublished: %w", err)
	}

	// 5. Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("service.PublishNoteToForum.Commit: %w", err)
	}

	return createdPost, nil
}
