package note

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

func (h *Handler) GetUserNotes(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}

	page, limit := utils.GetPageLimit(c)
	notes, total, err := h.service.ListUserNotes(c.Context(), userID, page, limit)
	if err != nil {
		log.Printf("Handler.GetUserNotes: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve notes"})
	}
	return c.Status(fiber.StatusOK).JSON(models.NewPaginatedResponse(notes, page, limit, total))
}

// GetUserNoteByID handles the request to fetch a single user note by its ID.
func (h *Handler) GetUserNoteByID(c *fiber.Ctx) error {
	// This is a protected route, so a user ID must exist in the context.
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}

	// Parse the note_id from the URL parameter into an int64.
	noteID, err := strconv.ParseInt(c.Params("note_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid note ID"})
	}

	note, err := h.service.GetUserNoteDetails(c.Context(), userID, noteID)
	if err != nil {
		// Check if the service returned a "not found" error.
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Note not found"})
		}
		// For all other errors, log them and return a generic server error.
		log.Printf("Handler.GetUserNoteByID: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve note"})
	}

	// If successful, return the note object with a 200 OK status.
	return c.Status(fiber.StatusOK).JSON(note)
}

func (h *Handler) CreateUserNote(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}

	var req models.CreateUserNoteData
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request: " + err.Error()})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	note, err := h.service.CreateUserNote(c.Context(), userID, req)
	if err != nil {
		log.Printf("Handler.CreateUserNote: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to create note"})
	}
	return c.Status(fiber.StatusCreated).JSON(note)
}

func (h *Handler) UpdateUserNote(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	noteID, err := strconv.ParseInt(c.Params("note_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid note ID"})
	}

	var req models.UpdateUserNoteData
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request: " + err.Error()})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	note, err := h.service.UpdateUserNote(c.Context(), userID, noteID, req)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Note not found or not owned by user"})
		}
		log.Printf("Handler.UpdateUserNote: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to update note"})
	}
	return c.Status(fiber.StatusOK).JSON(note)
}

func (h *Handler) DeleteUserNote(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	noteID, err := strconv.ParseInt(c.Params("note_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid note ID"})
	}

	err = h.service.DeleteUserNote(c.Context(), userID, noteID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Note not found or not owned by user"})
		}
		log.Printf("Handler.DeleteUserNote: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to delete note"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) AddLinkToNote(c *fiber.Ctx) error {
	noteID, err := strconv.ParseInt(c.Params("note_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid note ID"})
	}

	var req models.AddLinkToNoteData
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request: " + err.Error()})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	note, err := h.service.AddLinkToNote(c.Context(), noteID, req)
	if err != nil {
		log.Printf("Handler.AddLinkToNote: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to add link to note"})
	}
	return c.Status(fiber.StatusCreated).JSON(note)
}

func (h *Handler) RemoveLinkFromNote(c *fiber.Ctx) error {
	noteID, err := strconv.ParseInt(c.Params("note_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid note ID"})
	}
	linkID, err := strconv.ParseInt(c.Params("link_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid note ID"})
	}

	err = h.service.RemoveLinkFromNote(c.Context(), noteID, linkID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Link not found"})
		}
		log.Printf("Handler.RemoveLinkFromNote: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to remove link from note"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) PublishNoteToForum(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	noteID, err := strconv.ParseInt(c.Params("note_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid note ID"})
	}

	var req models.ForumPostPublishDetails
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request: " + err.Error()})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	forumPost, err := h.service.PublishNoteToForum(c.Context(), userID, noteID, req)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Note not found or not owned by user"})
		}
		if errors.Is(err, models.ErrConflict) {
			return c.Status(fiber.StatusConflict).JSON(models.ErrorResponse{Message: "Note already published"})
		}
		log.Printf("Handler.PublishNoteToForum: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to publish note to forum"})
	}
	return c.Status(fiber.StatusCreated).JSON(forumPost)
}
