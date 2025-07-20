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

// GetPostByID handles fetching a single post and its entire comment thread.
func (h *Handler) GetPostByID(c echo.Context) error {
	userID, _ := utils.GetUserIDFromContext(c)
	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid post ID"})
	}

	post, comments, err := h.service.GetPostDetails(c.Request().Context(), userID, postID)
	if err != nil {
		// Handle specific errors like not found
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve post details"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"post":     post,
		"comments": comments,
	})
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
		// Handle errors
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

	// The service needs to know the post_id. It should fetch it from the parent comment.
	// For now, we assume the service handles this lookup.
	// A more robust API might require the post_id in the request body or path.
	// Let's assume the service looks it up.
	comment, err := h.service.CreateComment(c.Request().Context(), userID, 0, &parentCommentID, req.Content) // Pass 0 for postID
	if err != nil {
		// Handle errors
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to create reply"})
	}
	return c.JSON(http.StatusCreated, comment)
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
