package note

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
	validate *validator.Validate
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
	}
}

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
	noteID, err := strconv.ParseInt(c.Param("note_id"), 10, 64)
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
	noteID, err := strconv.ParseInt(c.Param("note_id"), 10, 64)
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
	noteID, err := strconv.ParseInt(c.Param("note_id"), 10, 64)
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
	noteID, err := strconv.ParseInt(c.Param("note_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid note ID"})
	}
	linkID, err := strconv.ParseInt(c.Param("link_id"), 10, 64)
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
	noteID, err := strconv.ParseInt(c.Param("note_id"), 10, 64)
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
