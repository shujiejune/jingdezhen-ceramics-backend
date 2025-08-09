package forum

import (
	"errors"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/pkg/utils"
	"log"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service  ServiceInterface
	validate *validator.Validate
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
	}
}

func (h *Handler) GetPosts(c *fiber.Ctx) error {
	// A logged-in user might see personalized data (e.g., if they've liked a post)
	userID, _ := utils.GetUserIDFromContext(c)

	// Parse pagination and filter parameters from the URL query string.
	// e.g., /forum/posts?page=2&limit=20&sort=hottest&tag=glazing&category=1
	page, limit := utils.GetPageLimit(c)
	categoryID, _ := strconv.ParseInt(c.Query("category"), 10, 64) // Ignores error if not a valid int

	filters := models.PostFilters{
		Page:       page,
		Limit:      limit,
		Sort:       c.Query("sort"), // e.g., "hottest"
		TagName:    c.Query("tag"),  // e.g., "glazing"
		CategoryID: categoryID,      // Will be 0 if not provided or invalid
	}

	// Call the service with the filters.
	posts, total, err := h.service.GetPosts(c.Context(), userID, filters)
	if err != nil {
		log.Printf("Handler.GetPosts: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve posts"})
	}

	return c.Status(fiber.StatusOK).JSON(models.NewPaginatedResponse(posts, page, limit, total))
}

// GetPostByID handles fetching a single post and its entire comment thread.
func (h *Handler) GetPostByID(c *fiber.Ctx) error {
	userID, _ := utils.GetUserIDFromContext(c)
	postID, err := strconv.ParseInt(c.Params("post_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid post ID"})
	}

	post, comments, err := h.service.GetPostDetails(c.Context(), userID, postID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Post not found"})
		}
		log.Printf("Handler.GetPostByID: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve post details"})
	}

	return c.Status(fiber.StatusOK).JSON(map[string]any{
		"post":     post,
		"comments": comments,
	})
}

// SearchPosts performs a keyword search for forum posts.
func (h *Handler) SearchPosts(c *fiber.Ctx) error {
	userID, _ := utils.GetUserIDFromContext(c)
	page, limit := utils.GetPageLimit(c)
	query := c.Query("q")

	if len(query) < 3 {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Search query must be at least 3 characters long"})
	}

	// For a real-world application, this service method would ideally use a dedicated
	// search engine (like Elasticsearch) or PostgreSQL's Full-Text Search for better results.
	// A simple repository implementation will use a LIKE query.
	posts, total, err := h.service.SearchPosts(c.Context(), userID, query, page, limit)
	if err != nil {
		log.Printf("Handler.SearchPosts: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to search for posts"})
	}

	return c.Status(fiber.StatusOK).JSON(models.NewPaginatedResponse(posts, page, limit, total))
}

// GetTopicsTagCloud retrieves a list of tags, often sized by popularity for a tag cloud UI.
func (h *Handler) GetTopicsTagCloud(c *fiber.Ctx) error {
	tags, err := h.service.GetTags(c.Context())
	if err != nil {
		log.Printf("Handler.GetTopicsTagCloud: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve topics"})
	}
	return c.Status(fiber.StatusOK).JSON(tags)
}

// GetCategories retrieves a list of all forum categories.
func (h *Handler) GetCategories(c *fiber.Ctx) error {
	categories, err := h.service.GetCategories(c.Context())
	if err != nil {
		log.Printf("Handler.GetCategories: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve categories"})
	}
	return c.Status(fiber.StatusOK).JSON(categories)
}

// --- Protected Handlers ---

func (h *Handler) GetSavedForumPosts(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}

	page, limit := utils.GetPageLimit(c)
	savedForumPosts, total, err := h.service.GetSavedForumPosts(c.Context(), userID, page, limit)
	if err != nil {
		log.Printf("Handler.GetSavedForumPosts: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve saved forum posts"})
	}
	return c.Status(fiber.StatusOK).JSON(models.NewPaginatedResponse(savedForumPosts, page, limit, total))
}

// CreatePost handles creating a new forum post.
func (h *Handler) CreatePost(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}

	var req models.CreatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	post, err := h.service.CreatePost(c.Context(), userID, req)
	if err != nil {
		// Handle specific errors, e.g., invalid category ID
		if errors.Is(err, models.ErrInvalidForumPostCategoryID) {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid category for the post"})
		}
		log.Printf("Handler.CreatePost: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to create post"})
	}
	return c.Status(fiber.StatusCreated).JSON(post)
}

// UpdatePost handles updating an existing forum post.
func (h *Handler) UpdatePost(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	postID, err := strconv.ParseInt(c.Params("post_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid post ID"})
	}

	var req models.UpdatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	post, err := h.service.UpdatePost(c.Context(), userID, postID, req)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "You do not have permission to edit this post"})
		}
		log.Printf("Handler.UpdatePost: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to update post"})
	}
	return c.Status(fiber.StatusOK).JSON(post)
}

// DeletePost handles deleting a post.
func (h *Handler) DeletePost(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	userRole := c.Locals("userRole").(string) // Assumes JWT middleware sets this

	postID, err := strconv.ParseInt(c.Params("post_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid post ID"})
	}

	err = h.service.DeletePost(c.Context(), userID, userRole, postID)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "You do not have permission to delete this post"})
		}
		// Handle other errors
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to delete post"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// CreateComment handles both top-level and nested comments.
// It decides which based on the endpoint used.
func (h *Handler) CreateComment(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}

	postID, err := strconv.ParseInt(c.Params("post_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid post ID"})
	}

	var req models.CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	// This is for a top-level comment, so parentCommentID is nil.
	comment, err := h.service.CreateComment(c.Context(), userID, postID, nil, req.Content)
	if err != nil {
		log.Printf("Handler.CreateComment: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to create comment"})
	}
	return c.Status(fiber.StatusCreated).JSON(comment)
}

// CreateReply handles creating a nested comment.
// This would be mapped to POST /forum/comments/:comment_id/replies
func (h *Handler) CreateReply(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}

	postID, err := strconv.ParseInt(c.Params("post_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid post ID"})
	}

	parentCommentID, err := strconv.ParseInt(c.Params("comment_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid parent comment ID"})
	}

	var req models.CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	comment, err := h.service.CreateComment(c.Context(), userID, postID, &parentCommentID, req.Content)
	if err != nil {
		log.Printf("Handler.CreateComment: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to create reply"})
	}
	return c.Status(fiber.StatusCreated).JSON(comment)
}

// UpdateComment handles updating a comment.
func (h *Handler) UpdateComment(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	commentID, err := strconv.ParseInt(c.Params("comment_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid comment ID"})
	}

	var req models.CreateCommentRequest // Reusing this struct for content
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	comment, err := h.service.UpdateComment(c.Context(), userID, commentID, req.Content)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "You do not have permission to edit this comment"})
		}
		log.Printf("Handler.UpdateComment: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to update comment"})
	}
	return c.Status(fiber.StatusOK).JSON(comment)
}

// DeleteComment handles deleting a comment.
func (h *Handler) DeleteComment(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	userRole := c.Locals("userRole").(string)
	commentID, err := strconv.ParseInt(c.Params("comment_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid comment ID"})
	}

	if err := h.service.DeleteComment(c.Context(), userID, userRole, commentID); err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "You do not have permission to delete this comment"})
		}
		log.Printf("Handler.DeleteComment: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to delete comment"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// LikePost handles toggling a like on a post.
func (h *Handler) TogglePostLike(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	postID, err := strconv.ParseInt(c.Params("post_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid post ID"})
	}

	// The service will handle the toggle logic (like if not liked, unlike if liked).
	// We'll assume the service returns the new state.
	result, err := h.service.TogglePostLike(c.Context(), userID, postID)
	if err != nil {
		log.Printf("Handler.TogglePostLike: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to process like"})
	}
	return c.Status(fiber.StatusOK).JSON(result)
}

// SavePost handles toggling a save/bookmark on a post.
func (h *Handler) TogglePostSave(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	postID, err := strconv.ParseInt(c.Params("post_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid post ID"})
	}

	result, err := h.service.TogglePostSave(c.Context(), userID, postID)
	if err != nil {
		log.Printf("Handler.TogglePostSave: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to process save"})
	}
	return c.Status(fiber.StatusOK).JSON(result)
}

// LikeComment handles toggling a like on a comment.
func (h *Handler) ToggleCommentLike(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	commentID, err := strconv.ParseInt(c.Params("comment_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid comment ID"})
	}

	result, err := h.service.ToggleCommentLike(c.Context(), userID, commentID)
	if err != nil {
		log.Printf("Handler.ToggleCommentLike: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to process like"})
	}
	return c.Status(fiber.StatusOK).JSON(result)
}
