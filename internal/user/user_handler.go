package user

import (
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/pkg/utils"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service  ServiceInterface
	validate *validator.Validate // For request body validation
}

// NewHandler creates a new user handler.
// The AdminHandler can be this same handler, with routes protected by AdminRequired middleware.
func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
	}
}

// --- User Profile Routes ---
func (h *Handler) GetProfile(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	user, err := h.service.GetUserProfile(c.Request().Context(), userID)
	if err != nil {
		if err == models.ErrNotFound {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "User profile not found"})
		}
		c.Logger().Error("Handler.GetProfile: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve profile"})
	}
	return c.JSON(http.StatusOK, user)
}

func (h *Handler) UpdateProfile(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	var req models.UserUpdateData
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body: " + err.Error()})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	user, err := h.service.UpdateUserProfile(c.Request().Context(), userID, req)
	if err != nil {
		if err == models.ErrNotFound {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "User profile not found"})
		}
		c.Logger().Error("Handler.UpdateProfile: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to update profile"})
	}
	return c.JSON(http.StatusOK, user)
}

func (h *Handler) SubmitContactForm(c echo.Context) error {
	var req models.ContactFormData
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body: " + err.Error()})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	err := h.service.HandleContactSubmission(c.Request().Context(), req)
	if err != nil {
		c.Logger().Error("Handler.SubmitContactForm: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to submit contact form"})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Contact form submitted successfully"})
}

// --- User Notes Routes (within /profile group) ---
func (h *Handler) GetUserNotes(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	page, limit := utils.GetPageLimit(c)
	notes, total, err := h.service.ListUserNotes(c.Request().Context(), userID, page, limit)
	if err != nil {
		c.Logger().Error("Handler.GetUserNotes: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve notes"})
	}
	return c.JSON(http.StatusOK, models.NewPaginatedResponse(notes, page, limit, total))
}

func (h *Handler) CreateUserNote(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	var req models.CreateUserNoteData
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request: " + err.Error()})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	note, err := h.service.CreateUserNote(c.Request().Context(), userID, req)
	if err != nil {
		c.Logger().Error("Handler.CreateUserNote: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to create note"})
	}
	return c.JSON(http.StatusCreated, note)
}

func (h *Handler) UpdateUserNote(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	noteID, err := strconv.Atoi(c.Param("note_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid note ID"})
	}

	var req models.UpdateUserNoteData
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request: " + err.Error()})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	note, err := h.service.UpdateUserNote(c.Request().Context(), userID, noteID, req)
	if err != nil {
		if err == models.ErrNotFound {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Note not found or not owned by user"})
		}
		c.Logger().Error("Handler.UpdateUserNote: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to update note"})
	}
	return c.JSON(http.StatusOK, note)
}

func (h *Handler) DeleteUserNote(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	noteID, err := strconv.Atoi(c.Param("note_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid note ID"})
	}

	err = h.service.DeleteUserNote(c.Request().Context(), userID, noteID)
	if err != nil {
		if err == models.ErrNotFound {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Note not found or not owned by user"})
		}
		c.Logger().Error("Handler.DeleteUserNote: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to delete note"})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) AddLinkToNote(c echo.Context) error {
	noteID, err := strconv.Atoi(c.Param("note_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid note ID"})
	}

	var req models.AddLinkToNoteData
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request: " + err.Error()})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	note, err := h.service.AddLinkToNote(c.Request().Context(), noteID, req)
	if err != nil {
		c.Logger().Error("Handler.AddLinkToNote: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to add link to note"})
	}
	return c.JSON(http.StatusCreated, note)
}

func (h *Handler) RemoveLinkFromNote(c echo.Context) error {
	noteID, err := strconv.Atoi(c.Param("note_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid note ID"})
	}
	linkID, err := strconv.Atoi(c.Param("link_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid note ID"})
	}

	err = h.service.RemoveLinkFromNote(c.Request().Context(), noteID, linkID)
	if err != nil {
		if err == models.ErrNotFound {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Link not found"})
		}
		c.Logger().Error("Handler.RemoveLinkFromNote: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to remove link from note"})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) PublishNoteToForum(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	noteID, err := strconv.Atoi(c.Param("note_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid note ID"})
	}

	var req models.ForumPostPublishDetails
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request: " + err.Error()})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	forumPost, err := h.service.PublishNoteToForum(c.Request().Context(), userID, noteID, req)
	if err != nil {
		if err == models.ErrNotFound {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Note not found or not owned by user"})
		}
		if err == models.ErrConflict {
			return c.JSON(http.StatusConflict, models.ErrorResponse{Message: "Note already published"})
		}
		c.Logger().Error("Handler.PublishNoteToForum: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to publish note to forum"})
	}
	return c.JSON(http.StatusCreated, forumPost)
}

// GetNotifications would be similar to GetUserNotes, fetching from a notification service/repo
func (h *Handler) GetNotifications(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	page, limit := utils.GetPageLimit(c)
	notifications, total, err := h.service.GetNotifications(c.Request().Context(), userID, page, limit)
	if err != nil {
		c.Logger().Error("Handler.GetNotifications: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve notifications"})
	}
	return c.JSON(http.StatusOK, models.NewPaginatedResponse(notifications, page, limit, total))
}

// GetFavoriteArtworks - requires gallery service/repo interaction
func (h *Handler) GetFavoriteArtworks(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	page, limit := utils.GetPageLimit(c)
	favArtworks, total, err := h.service.GetFavArtworks(c.Request().Context(), userID, page, limit)
	if err != nil {
		c.Logger().Error("Handler.GetFavArtworks: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve favorite artworks"})
	}
	return c.JSON(http.StatusOK, models.NewPaginatedResponse(favArtworks, page, limit, total))
}

// GetSavedForumPosts - requires forum service/repo interaction
func (h *Handler) GetSavedForumPosts(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	page, limit := utils.GetPageLimit(c)
	savedForumPosts, total, err := h.service.GetSavedForumPosts(c.Request().Context(), userID, page, limit)
	if err != nil {
		c.Logger().Error("Handler.GetSavedForumPosts: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve saved forum posts"})
	}
	return c.JSON(http.StatusOK, models.NewPaginatedResponse(savedForumPosts, page, limit, total))
}

// --- Admin User Management Routes ---
// These methods are part of the same *user.Handler but will be protected by AdminRequired middleware in router.go
func (h *Handler) AdminListUsers(c echo.Context) error {
	page, limit := utils.GetPageLimit(c)
	users, total, err := h.service.AdminListUsers(c.Request().Context(), page, limit)
	if err != nil {
		c.Logger().Error("Handler.AdminListUsers: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to list users"})
	}
	return c.JSON(http.StatusOK, models.NewPaginatedResponse(users, page, limit, total))
}

func (h *Handler) AdminUpdateUserRole(c echo.Context) error {
	targetUserID := c.Param("user_id")
	var req struct {
		Role string `json:"role" validate:"required,oneof=admin normal_user"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body: " + err.Error()})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	err := h.service.AdminUpdateUserRole(c.Request().Context(), targetUserID, req.Role)
	if err != nil {
		if err == models.ErrNotFound {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Target user not found"})
		}
		c.Logger().Error("Handler.AdminUpdateUserRole: ", err)
		// Check for specific service errors if any (e.g., invalid role error from service)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to update user role"})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "User role updated successfully"})
}
