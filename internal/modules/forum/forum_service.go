package forum

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"log"
)

type ServiceInterface interface {
	GetPosts(ctx context.Context, userID string, filters models.PostFilters) ([]models.ForumPost, int, error)
	GetPostDetails(ctx context.Context, userID string, postID int64) (*models.ForumPost, []*models.ForumComment, error)
	SearchPosts(ctx context.Context, userID, query string, page, limit int) ([]models.ForumPost, int, error)
	GetCategories(ctx context.Context) ([]models.ForumCategory, error)
	GetTags(ctx context.Context) ([]models.Tag, error)

	CreatePost(ctx context.Context, userID string, data models.CreatePostRequest) (*models.ForumPost, error)
	UpdatePost(ctx context.Context, userID string, postID int64, data models.UpdatePostRequest) (*models.ForumPost, error)
	DeletePost(ctx context.Context, userID, userRole string, postID int64) error
	CreateComment(ctx context.Context, userID string, postID int64, parentCommentID *int64, content string) (*models.ForumComment, error)
	UpdateComment(ctx context.Context, userID string, commentID int64, content string) (*models.ForumComment, error)
	DeleteComment(ctx context.Context, userID, userRole string, commentID int64) error

	TogglePostLike(ctx context.Context, userID string, postID int64) (*models.ToggleResult, error)
	TogglePostSave(ctx context.Context, userID string, postID int64) (*models.ToggleResult, error)
	ToggleCommentLike(ctx context.Context, userID string, commentID int64) (*models.ToggleResult, error)
}

type Service struct {
	repo RepositoryInterface
}

func NewService(repo RepositoryInterface) ServiceInterface {
	return &Service{repo: repo}
}

// --- Private Helper Functions ---

// enrichPostsWithUserStatus fetches like/save status for a slice of posts for a given user.
func (s *Service) enrichPostsWithUserStatus(ctx context.Context, userID string, posts []models.ForumPost) {
	if len(posts) == 0 || userID == "" {
		return
	}
	postIDs := make([]int64, len(posts))
	for i, p := range posts {
		postIDs[i] = p.ID
	}

	likedMap, err := s.repo.CheckPostsLikedByUser(ctx, userID, postIDs)
	if err != nil {
		log.Printf("WARN: could not check post likes for user %s: %v", userID, err)
	}
	savedMap, err := s.repo.CheckPostsSavedByUser(ctx, userID, postIDs)
	if err != nil {
		log.Printf("WARN: could not check post saves for user %s: %v", userID, err)
	}

	for i := range posts {
		if likedMap[posts[i].ID] {
			posts[i].IsLikedByMe = true
		}
		if savedMap[posts[i].ID] {
			posts[i].IsSavedByMe = true
		}
	}
}

// enrichCommentsWithUserStatus fetches like status for a slice of comments for a given user.
func (s *Service) enrichCommentsWithUserStatus(ctx context.Context, userID string, comments []models.ForumComment) {
	if len(comments) == 0 || userID == "" {
		return
	}
	commentIDs := make([]int64, len(comments))
	for i, c := range comments {
		commentIDs[i] = c.ID
	}

	likedMap, err := s.repo.CheckCommentsLikedByUser(ctx, userID, commentIDs)
	if err != nil {
		log.Printf("WARN: could not check comment likes for user %s: %v", userID, err)
	}

	for i := range comments {
		if likedMap[comments[i].ID] {
			comments[i].IsLikedByMe = true
		}
	}
}

// GetPosts retrieves a list of posts, enriching it with user-specific data.
func (s *Service) GetPosts(ctx context.Context, userID string, filters models.PostFilters) ([]models.ForumPost, int, error) {
	posts, total, err := s.repo.FindAllPosts(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("service.GetPosts: %w", err)
	}

	if len(posts) > 0 && userID != "" {
		s.enrichPostsWithUserStatus(ctx, userID, posts)
	}

	return posts, total, nil
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

	// Enrich with user-specific data (likes/saves) if a user is logged in.
	if userID != "" {
		s.enrichPostsWithUserStatus(ctx, userID, []models.ForumPost{*post})
		if len(flatComments) > 0 {
			s.enrichCommentsWithUserStatus(ctx, userID, flatComments)
		}
	}

	// Build the nested comment tree from the flat list
	commentMap := make(map[int64]*models.ForumComment)
	for i := range flatComments {
		comment := flatComments[i]
		commentMap[comment.ID] = &comment
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

	return post, nestedComments, nil
}

// SearchPosts performs a search and enriches results with user-specific data.
func (s *Service) SearchPosts(ctx context.Context, userID, query string, page, limit int) ([]models.ForumPost, int, error) {
	posts, total, err := s.repo.SearchPosts(ctx, query, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("service.SearchPosts: %w", err)
	}

	if len(posts) > 0 && userID != "" {
		s.enrichPostsWithUserStatus(ctx, userID, posts)
	}

	return posts, total, nil
}

func (s *Service) GetCategories(ctx context.Context) ([]models.ForumCategory, error) {
	return s.repo.FindAllCategories(ctx)
}

func (s *Service) GetTags(ctx context.Context) ([]models.Tag, error) {
	return s.repo.FindAllTags(ctx)
}

// CreatePost handles the business logic for creating a new post.
func (s *Service) CreatePost(ctx context.Context, userID string, data models.CreatePostRequest) (*models.ForumPost, error) {
	// Business logic: Validate that the CategoryID exists.
	// This would require a `FindCategoryByID` method in the repository.
	// For now, we'll let the database foreign key constraint handle it.
	categoryID := data.CategoryID
	if _, err := s.repo.FindCategoryByID(ctx, categoryID); err != nil {
		return nil, fmt.Errorf("service.CreatePost.FindCategory: %w", err)
	}

	return s.repo.CreatePost(ctx, userID, data)
}

// UpdatePost handles business logic for updating a post, including authorization.
func (s *Service) UpdatePost(ctx context.Context, userID string, postID int64, data models.UpdatePostRequest) (*models.ForumPost, error) {
	isOwner, err := s.repo.CheckPostOwnership(ctx, postID, userID)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		return nil, models.ErrForbidden
	}
	return s.repo.UpdatePost(ctx, postID, data)
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

// CreateComment handles creation of both top-level and nested comments.
func (s *Service) CreateComment(ctx context.Context, userID string, postID int64, parentCommentID *int64, content string) (*models.ForumComment, error) {
	// Business logic:
	// 1. Check if post exists
	_, err := s.repo.FindPostByID(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("service.CreateComment.FindPost: %w", err)
	}

	// 2. If parentCommentID is not nil, check if the parent comment exists and belongs to the same post
	if parentCommentID != nil {
		parentComment, err := s.repo.FindCommentByID(ctx, *parentCommentID)
		if err != nil {
			return nil, fmt.Errorf("parent comment not found: %w", err)
		}
		if parentComment.PostID != postID {
			return nil, models.ErrForbidden
		}
	}

	// 3. Save the comment to the database
	savedComment, err := s.repo.CreateComment(ctx, userID, postID, parentCommentID, content)
	if err != nil {
		return nil, err
	}

	// 4. After success, send notification to the original author
	go func() {
		// Use context.Background() for background tasks
		s.notificationSvc.SendNewCommentNotification(context.Background(), savedComment)
		log.Printf("INFO: Triggered notification for new comment %d", savedComment.ID)
	}()

	return savedComment, nil
}

// UpdateComment handles business logic for updating a comment, including authorization.
func (s *Service) UpdateComment(ctx context.Context, userID string, commentID int64, content string) (*models.ForumComment, error) {
	isOwner, err := s.repo.CheckCommentOwnership(ctx, commentID, userID)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		return nil, models.ErrForbidden
	}
	return s.repo.UpdateComment(ctx, commentID, content)
}

// DeleteComment handles business logic for deleting a comment, including authorization.
func (s *Service) DeleteComment(ctx context.Context, userID, userRole string, commentID int64) error {
	if userRole != models.RoleAdmin {
		isOwner, err := s.repo.CheckCommentOwnership(ctx, commentID, userID)
		if err != nil {
			return err
		}
		if !isOwner {
			return models.ErrForbidden
		}
	}
	return s.repo.DeleteComment(ctx, commentID)
}

// TogglePostLike handles the logic to add or remove a like from a post.
func (s *Service) TogglePostLike(ctx context.Context, userID string, postID int64) (*models.ToggleResult, error) {
	isLiked, err := s.repo.IsPostLikedByUser(ctx, userID, postID)
	if err != nil {
		return nil, err
	}

	var newCount int
	if isLiked {
		newCount, err = s.repo.RemoveLikeFromPost(ctx, userID, postID)
	} else {
		newCount, err = s.repo.AddLikeToPost(ctx, userID, postID)
	}
	if err != nil {
		return nil, err
	}

	return &models.ToggleResult{IsActive: !isLiked, NewCount: newCount}, nil
}

// TogglePostSave handles the logic to add or remove a save from a post.
func (s *Service) TogglePostSave(ctx context.Context, userID string, postID int64) (*models.ToggleResult, error) {
	isSaved, err := s.repo.IsPostSavedByUser(ctx, userID, postID)
	if err != nil {
		return nil, err
	}

	if isSaved {
		err = s.repo.RemoveSaveFromPost(ctx, userID, postID)
	} else {
		err = s.repo.AddSaveForPost(ctx, userID, postID)
	}
	if err != nil {
		return nil, err
	}

	// Save count is not typically shown, so we can return 0.
	return &models.ToggleResult{IsActive: !isSaved, NewCount: 0}, nil
}

// ToggleCommentLike handles the logic to add or remove a like from a comment.
func (s *Service) ToggleCommentLike(ctx context.Context, userID string, commentID int64) (*models.ToggleResult, error) {
	isLiked, err := s.repo.IsCommentLikedByUser(ctx, userID, commentID)
	if err != nil {
		return nil, err
	}

	var newCount int
	if isLiked {
		newCount, err = s.repo.RemoveLikeFromComment(ctx, userID, commentID)
	} else {
		newCount, err = s.repo.AddLikeToComment(ctx, userID, commentID)
	}
	if err != nil {
		return nil, err
	}

	return &models.ToggleResult{IsActive: !isLiked, NewCount: newCount}, nil
}
