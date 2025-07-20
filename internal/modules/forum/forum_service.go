package forum

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
)

type ServiceInterface interface {
	GetPosts(ctx context.Context, userID string, filters models.PostFilters) ([]models.ForumPost, int, error)
	SearchPosts(ctx context.Context, userID, query string, page, limit int) ([]models.ForumPost, int, error)
	GetPostDetails(ctx context.Context, userID string, postID int64) (*models.ForumPost, []*models.ForumComment, error)
	GetCategories(ctx context.Context) ([]models.ForumCategory, error)
	GetTags(ctx context.Context) ([]models.Tag, error)
	CreatePost(ctx context.Context, userID string, data models.CreatePostRequest) (*models.ForumPost, error)
	UpdatePost(ctx context.Context, userID string, postID int64, data models.UpdatePostRequest) (*models.ForumPost, error)
	DeletePost(ctx context.Context, userID, userRole string, postID int64) error
	CreateComment(ctx context.Context, userID string, postID int64, parentCommentID *int64, content string) (*models.ForumComment, error)
	UpdateComment(ctx context.Context, userID string, commentID int64, content string) (*models.ForumComment, error)
	DeleteComment(ctx context.Context, userID, userRole string, commentID int64) error
	LikePost(ctx context.Context, userID string, postID int64, like bool) error
	SavePost(ctx context.Context, userID string, postID int64, save bool) error
	LikeComment(ctx context.Context, userID string, commentID int64, like bool) error
}

type Service struct {
	repo RepositoryInterface
}

func NewService(repo RepositoryInterface) ServiceInterface {
	return &Service{repo: repo}
}

// GetPostDetails fetches a post and its full comment thread, structured for the client.
func (s *Service) GetPostDetails(ctx context.Context, userID string, postID int64) (*models.ForumPost, []*models.ForumComment, error) {
	post, err := s.repo.FindPostByID(ctx, postID)
	if err != nil {
		return nil, nil, fmt.Errorf("service.GetPostDetails.FindPost: %w", err)
	}

	// Fetch all comments as a flat list, correctly sorted by the repository
	flatComments, err := s.repo.FindCommentsByPostID(ctx, postID)
	if err != nil {
		return nil, nil, fmt.Errorf("service.GetPostDetails.FindComments: %w", err)
	}

	// Business logic to build the nested comment tree from the flat list
	commentMap := make(map[int64]*models.ForumComment)
	for i := range flatComments {
		commentMap[flatComments[i].ID] = &flatComments[i]
	}

	var nestedComments []*models.ForumComment
	for _, comment := range flatComments {
		if comment.ParentCommentID == nil {
			nestedComments = append(nestedComments, commentMap[comment.ID])
		} else {
			if parent, ok := commentMap[*comment.ParentCommentID]; ok {
				parent.Replies = append(parent.Replies, commentMap[comment.ID])
			}
		}
	}

	// TODO: Check user's likes/saves for the post and comments if userID is provided

	return post, nestedComments, nil
}

// CreateComment handles creation of both top-level and nested comments.
func (s *Service) CreateComment(ctx context.Context, userID string, postID int64, parentCommentID *int64, content string) (*models.ForumComment, error) {
	// Business logic:
	// 1. Check if post exists
	// 2. If parentCommentID is not nil, check if the parent comment exists and belongs to the same post
	// For simplicity, we'll let DB foreign key constraints handle this.
	return s.repo.CreateComment(ctx, userID, postID, parentCommentID, content)
}

// DeletePost checks for ownership or admin role before deleting.
func (s *Service) DeletePost(ctx context.Context, userID, userRole string, postID int64) error {
	if userRole != models.RoleAdmin {
		isOwner, err := s.repo.CheckPostOwnership(ctx, postID, userID)
		if err != nil {
			return err
		}
		if !isOwner {
			return models.ErrForbidden
		}
	}
	return s.repo.DeletePost(ctx, postID)
}
