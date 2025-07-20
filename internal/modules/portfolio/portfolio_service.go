package portfolio

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
)

// ServiceInterface defines the methods for portfolio business logic.
type ServiceInterface interface {
	GetWorks(ctx context.Context, userID string, filters models.PortfolioFilters) ([]models.PortfolioWork, int, error)
	GetWorkByID(ctx context.Context, userID string, workID int64) (*models.PortfolioWork, error)
	CreateWork(ctx context.Context, userID string, data models.CreateWorkRequest) (*models.PortfolioWork, error)
	UpdateWork(ctx context.Context, userID string, workID int64, data models.UpdateWorkRequest) (*models.PortfolioWork, error)
	DeleteWork(ctx context.Context, userID, userRole string, workID int64) error
	LeaveKudo(ctx context.Context, userID string, workID int64) (int, error)
}

// Service provides business logic for the portfolio module.
type Service struct {
	repo RepositoryInterface
}

// NewService creates a new portfolio service.
func NewService(repo RepositoryInterface) ServiceInterface {
	return &Service{repo: repo}
}

// GetWorks retrieves a list of works, enriching it with user-specific data.
func (s *Service) GetWorks(ctx context.Context, userID string, filters models.PortfolioFilters) ([]models.PortfolioWork, int, error) {
	works, total, err := s.repo.FindAllWorks(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("service.GetWorks: %w", err)
	}

	if len(works) > 0 && userID != "" {
		workIDs := make([]int64, len(works))
		for i, work := range works {
			workIDs[i] = work.ID
		}
		kudoMap, err := s.repo.CheckKudos(ctx, userID, workIDs)
		if err != nil {
			// Log but don't fail the entire request, as kudos status is non-critical.
			fmt.Printf("WARN: could not check kudos for user %s: %v\n", userID, err)
		}
		for i := range works {
			if kudoMap[works[i].ID] {
				works[i].HasMyKudo = true
			}
		}
	}
	return works, total, nil
}

// GetWorkByID retrieves a single work and all its related data.
func (s *Service) GetWorkByID(ctx context.Context, userID string, workID int64) (*models.PortfolioWork, error) {
	work, err := s.repo.FindWorkByID(ctx, workID)
	if err != nil {
		return nil, fmt.Errorf("service.GetWorkByID: %w", err)
	}

	// Fetch and attach related data, logging non-critical errors.
	images, err := s.repo.GetWorkImages(ctx, workID)
	if err != nil {
		fmt.Printf("WARN: could not get images for work %d: %v\n", workID, err)
	}
	work.Images = images

	tags, err := s.repo.GetWorkTags(ctx, workID)
	if err != nil {
		fmt.Printf("WARN: could not get tags for work %d: %v\n", workID, err)
	}
	work.Tags = tags

	if userID != "" {
		kudoMap, err := s.repo.CheckKudos(ctx, userID, []int64{workID})
		if err != nil {
			fmt.Printf("WARN: could not check kudo for work %d, user %s: %v\n", workID, userID, err)
		}
		if kudoMap[workID] {
			work.HasMyKudo = true
		}
	}
	return work, nil
}

// CreateWork handles the business logic for creating a new portfolio work.
func (s *Service) CreateWork(ctx context.Context, userID string, data models.CreateWorkRequest) (*models.PortfolioWork, error) {
	// Business logic could be added here, e.g., checking if a user has reached a submission limit.
	return s.repo.CreateWork(ctx, userID, data)
}

// UpdateWork handles the business logic for updating a portfolio work, including ownership check.
func (s *Service) UpdateWork(ctx context.Context, userID string, workID int64, data models.UpdateWorkRequest) (*models.PortfolioWork, error) {
	isOwner, err := s.repo.CheckWorkOwnership(ctx, workID, userID)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		return nil, models.ErrForbidden
	}
	return s.repo.UpdateWork(ctx, workID, data)
}

// DeleteWork handles the business logic for deleting a portfolio work, including ownership/admin check.
func (s *Service) DeleteWork(ctx context.Context, userID, userRole string, workID int64) error {
	if userRole != models.RoleAdmin {
		isOwner, err := s.repo.CheckWorkOwnership(ctx, workID, userID)
		if err != nil {
			return err
		}
		if !isOwner {
			return models.ErrForbidden
		}
	}
	return s.repo.DeleteWork(ctx, workID)
}

// LeaveKudo handles the business logic for a user leaving a kudo on a work.
func (s *Service) LeaveKudo(ctx context.Context, userID string, workID int64) (int, error) {
	_, err := s.repo.FindWorkByID(ctx, workID)
	if err != nil {
		return 0, err // Ensure work exists before trying to kudo it.
	}
	return s.repo.AddKudo(ctx, userID, workID)
}
