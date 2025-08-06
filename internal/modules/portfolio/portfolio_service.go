package portfolio

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"log"
)

const (
	// Define the business rule for submission limits.
	maxPortfolioWorksPerUser = 20
)

// ServiceInterface defines the methods for portfolio business logic.
type ServiceInterface interface {
	GetWorks(ctx context.Context, userID string, filters models.PortfolioFilters) ([]models.PortfolioWork, int, error)
	GetWorkByID(ctx context.Context, userID string, workID int64) (*models.PortfolioWork, error)
	CreateWork(ctx context.Context, userID string, data models.CreateWorkRequest) (*models.PortfolioWork, error)
	UpdateWork(ctx context.Context, userID string, workID int64, data models.UpdateWorkRequest) (*models.PortfolioWork, error)
	DeleteWork(ctx context.Context, userID, userRole string, workID int64) error
	ToggleWorkUpvote(ctx context.Context, userID string, workID int64) (*models.ToggleUpvoteResult, error)
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
		upvoteMap, err := s.repo.CheckUpvotes(ctx, userID, workIDs)
		if err != nil {
			// Log but don't fail the entire request, as upvotes status is non-critical.
			fmt.Printf("WARN: could not check upvotes for user %s: %v\n", userID, err)
		}
		for i := range works {
			if upvoteMap[works[i].ID] {
				works[i].UpvotedByMe = true
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
		upvoted, err := s.repo.IsWorkUpvotedByUser(ctx, userID, workID)
		if err != nil {
			log.Printf("WARN: could not check upvote for work %d, user %s: %v\n", workID, userID, err)
		}
		work.UpvotedByMe = upvoted
	}
	return work, nil
}

// CreateWork handles the business logic for creating a new portfolio work.
func (s *Service) CreateWork(ctx context.Context, userID string, data models.CreateWorkRequest) (*models.PortfolioWork, error) {
	// Check if the user has reached their submission limit.
	count, err := s.repo.GetWorkCountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service.CreateWork.GetWorkCount: %w", err)
	}
	if count >= maxPortfolioWorksPerUser {
		return nil, models.ErrLimitExceeded // Return a specific application error
	}

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

// ToggleWorkUpvote handles the business logic for a user upvoting a work.
func (s *Service) ToggleWorkUpvote(ctx context.Context, userID string, workID int64) (*models.ToggleUpvoteResult, error) {
	isUpvoted, err := s.repo.IsWorkUpvotedByUser(ctx, userID, workID)
	if err != nil {
		return nil, err
	}

	var newCount int
	if isUpvoted {
		newCount, err = s.repo.Downvote(ctx, userID, workID)
	} else {
		newCount, err = s.repo.Upvote(ctx, userID, workID)
	}
	if err != nil {
		return nil, err
	}

	return &models.ToggleUpvoteResult{IsUpvoted: !isUpvoted, NewCount: newCount}, nil
}
