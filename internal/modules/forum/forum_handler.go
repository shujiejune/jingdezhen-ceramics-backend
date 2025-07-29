package forum

import (
	"errors"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/pkg/utils"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
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

func (h *Handler) GetPosts(c echo.Context) error {
	// A logged-in user might see personalized data (e.g., if they've liked a post)
	userID, _ := utils.GetUserIDFromContext(c)

	// Parse pagination and filter parameters from the URL query string.
	// e.g., /forum/posts?page=2&limit=20&sort=hottest&tag=glazing&category=1
	page, limit := utils.GetPageLimit(c)
	categoryID, _ := strconv.ParseInt(c.QueryParam("category"), 10, 64) // Ignores error if not a valid int

	filters := models.PostFilters{
		Page:       page,
		Limit:      limit,
		Sort:       c.QueryParam("sort"), // e.g., "hottest"
		Tag:        c.QueryParam("tag"),  // e.g., "glazing"
		CategoryID: categoryID,           // Will be 0 if not provided or invalid
	}

	// Call the service with the filters.
	posts, total, err := h.service.GetPosts(c.Request().Context(), userID, filters)
	if err != nil {
		c.Logger().Error("Handler.GetPosts: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve posts"})
	}

	return c.JSON(http.StatusOK, models.NewPaginatedResponse(posts, page, limit, total))
}

// GetPostByID handles fetching a single post and its entire comment thread.
func (h *Handler) GetPostByID(c echo.Context) error {
	userID, _ := utils.GetUserIDFromContext(c)
	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid post ID"})
	}

	post, comments, err := h.service.GetPostDetails(c.Request().Context(), userID, postID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Post not found"})
		}
		c.Logger().Error("Handler.GetPostByID: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve post details"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"post":     post,
		"comments": comments,
	})
}

// SearchPosts performs a keyword search for forum posts.
func (h *Handler) SearchPosts(c echo.Context) error {
	userID, _ := utils.GetUserIDFromContext(c)
	page, limit := utils.GetPageLimit(c)
	query := c.QueryParam("q")

	if len(query) < 3 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Search query must be at least 3 characters long"})
	}

	// For a real-world application, this service method would ideally use a dedicated
	// search engine (like Elasticsearch) or PostgreSQL's Full-Text Search for better results.
	// A simple repository implementation will use a LIKE query.
	posts, total, err := h.service.SearchPosts(c.Request().Context(), userID, query, page, limit)
	if err != nil {
		c.Logger().Error("Handler.SearchPosts: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to search for posts"})
	}

	return c.JSON(http.StatusOK, models.NewPaginatedResponse(posts, page, limit, total))
}

// GetTopicsTagCloud retrieves a list of tags, often sized by popularity for a tag cloud UI.
func (h *Handler) GetTopicsTagCloud(c echo.Context) error {
	tags, err := h.service.GetTags(c.Request().Context())
	if err != nil {
		c.Logger().Error("Handler.GetTopicsTagCloud: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve topics"})
	}
	return c.JSON(http.StatusOK, tags)
}

// GetCategories retrieves a list of all forum categories.
func (h *Handler) GetCategories(c echo.Context) error {
	categories, err := h.service.GetCategories(c.Request().Context())
	if err != nil {
		c.Logger().Error("Handler.GetCategories: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve categories"})
	}
	return c.JSON(http.StatusOK, categories)
}

// --- Protected Handlers ---

// CreatePost handles creating a new forum post.
func (h *Handler) CreatePost(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	var req models.CreatePostRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	post, err := h.service.CreatePost(c.Request().Context(), userID, req)
	if err != nil {
		// Handle specific errors, e.g., invalid category ID
		if errors.Is(err, models.ErrInvalidForumPostCategoryID) {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid category for the post"})
		}
		c.Logger().Error("Handler.CreatePost: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to create post"})
	}
	return c.JSON(http.StatusCreated, post)
}

// UpdatePost handles updating an existing forum post.
func (h *Handler) UpdatePost(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid post ID"})
	}

	var req models.UpdatePostRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	post, err := h.service.UpdatePost(c.Request().Context(), userID, postID, req)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{Message: "You do not have permission to edit this post"})
		}
		c.Logger().Error("Handler.UpdatePost: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to update post"})
	}
	return c.JSON(http.StatusOK, post)
}

// DeletePost handles deleting a post.
func (h *Handler) DeletePost(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	userRole := c.Get("userRole").(string) // Assumes JWT middleware sets this

	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid post ID"})
	}

	err = h.service.DeletePost(c.Request().Context(), userID, userRole, postID)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{Message: "You do not have permission to delete this post"})
		}
		// Handle other errors
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to delete post"})
	}
	return c.NoContent(http.StatusNoContent)
}

// CreateComment handles both top-level and nested comments.
// It decides which based on the endpoint used.
func (h *Handler) CreateComment(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid post ID"})
	}

	var req models.CreateCommentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	// This is for a top-level comment, so parentCommentID is nil.
	comment, err := h.service.CreateComment(c.Request().Context(), userID, postID, nil, req.Content)
	if err != nil {
		c.Logger().Error("Handler.CreateComment: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to create comment"})
	}
	return c.JSON(http.StatusCreated, comment)
}

// CreateReply handles creating a nested comment.
// This would be mapped to POST /forum/comments/:comment_id/replies
func (h *Handler) CreateReply(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid post ID"})
	}

	parentCommentID, err := strconv.ParseInt(c.Param("comment_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid parent comment ID"})
	}

	var req models.CreateCommentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	comment, err := h.service.CreateComment(c.Request().Context(), userID, postID, &parentCommentID, req.Content)
	if err != nil {
		c.Logger().Error("Handler.CreateComment: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to create reply"})
	}
	return c.JSON(http.StatusCreated, comment)
}

// UpdateComment handles updating a comment.
func (h *Handler) UpdateComment(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	commentID, err := strconv.ParseInt(c.Param("comment_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid comment ID"})
	}

	var req models.CreateCommentRequest // Reusing this struct for content
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	comment, err := h.service.UpdateComment(c.Request().Context(), userID, commentID, req.Content)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{Message: "You do not have permission to edit this comment"})
		}
		c.Logger().Error("Handler.UpdateComment: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to update comment"})
	}
	return c.JSON(http.StatusOK, comment)
}

// DeleteComment handles deleting a comment.
func (h *Handler) DeleteComment(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	userRole := c.Get("userRole").(string)
	commentID, err := strconv.ParseInt(c.Param("comment_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid comment ID"})
	}

	if err := h.service.DeleteComment(c.Request().Context(), userID, userRole, commentID); err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{Message: "You do not have permission to delete this comment"})
		}
		c.Logger().Error("Handler.DeleteComment: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to delete comment"})
	}
	return c.NoContent(http.StatusNoContent)
}

// LikePost handles toggling a like on a post.
func (h *Handler) TogglePostLike(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid post ID"})
	}

	// The service will handle the toggle logic (like if not liked, unlike if liked).
	// We'll assume the service returns the new state.
	result, err := h.service.TogglePostLike(c.Request().Context(), userID, postID)
	if err != nil {
		c.Logger().Error("Handler.TogglePostLike: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to process like"})
	}
	return c.JSON(http.StatusOK, result)
}

// SavePost handles toggling a save/bookmark on a post.
func (h *Handler) TogglePostSave(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid post ID"})
	}

	result, err := h.service.TogglePostSave(c.Request().Context(), userID, postID)
	if err != nil {
		c.Logger().Error("Handler.TogglePostSave: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to process save"})
	}
	return c.JSON(http.StatusOK, result)
}

// LikeComment handles toggling a like on a comment.
func (h *Handler) ToggleCommentLike(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	commentID, err := strconv.ParseInt(c.Param("comment_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid comment ID"})
	}

	result, err := h.service.ToggleCommentLike(c.Request().Context(), userID, commentID)
	if err != nil {
		c.Logger().Error("Handler.ToggleCommentLike: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to process like"})
	}
	return c.JSON(http.StatusOK, result)
}
