package portfolio

import (
	"errors"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/pkg/utils"
	"log"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for the portfolio module.
type Handler struct {
	service  ServiceInterface
	validate *validator.Validate
}

// NewHandler creates a new portfolio handler.
func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
	}
}

// GetWorks handles requests to get a paginated list of portfolio works.
func (h *Handler) GetWorks(c *fiber.Ctx) error {
	userID, _ := utils.GetUserIDFromContext(c) // Optional: for upvotes status
	page, limit := utils.GetPageLimit(c)
	sort := c.Query("sort")
	tagsQuery := c.Query("tags")
	var tags []string
	if tagsQuery != "" {
		// Split the comma-separated string into a slice
		tags = strings.Split(tagsQuery, ",")
	}

	filters := models.PortfolioFilters{
		Page:  page,
		Limit: limit,
		Tags:  tags,
		Sort:  sort,
	}

	works, total, err := h.service.GetWorks(c.Context(), userID, filters)
	if err != nil {
		log.Printf("Handler.GetWorks: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve portfolio works"})
	}

	return c.Status(fiber.StatusOK).JSON(models.NewPaginatedResponse(works, page, limit, total))
}

// GetWorkByID handles requests to get a single, detailed portfolio work.
func (h *Handler) GetWorkByID(c *fiber.Ctx) error {
	userID, _ := utils.GetUserIDFromContext(c)
	workID, err := strconv.ParseInt(c.Params("work_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid work ID"})
	}

	work, err := h.service.GetWorkByID(c.Context(), userID, workID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Portfolio work not found"})
		}
		log.Printf("Handler.GetWorkByID: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve portfolio work"})
	}

	return c.Status(fiber.StatusOK).JSON(work)
}

// --- Protected Handlers ---

// CreateWork handles requests from authenticated users to create a new portfolio work.
func (h *Handler) CreateWork(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}

	var req models.CreateWorkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	work, err := h.service.CreateWork(c.Context(), userID, req)
	if err != nil {
		log.Printf("Handler.CreateWork: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to create portfolio work"})
	}
	return c.Status(fiber.StatusCreated).JSON(work)
}

// UpdateWork handles requests from authenticated users to update their own portfolio work.
func (h *Handler) UpdateWork(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	workID, err := strconv.ParseInt(c.Params("work_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid work ID"})
	}

	var req models.UpdateWorkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	work, err := h.service.UpdateWork(c.Context(), userID, workID, req)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "You do not have permission to edit this work"})
		}
		log.Printf("Handler.UpdateWork: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to update portfolio work"})
	}
	return c.Status(fiber.StatusOK).JSON(work)
}

// DeleteWork handles requests to delete a portfolio work, checking for ownership or admin role.
func (h *Handler) DeleteWork(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	userRole := c.Locals("userRole").(string)

	workID, err := strconv.ParseInt(c.Params("work_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid work ID"})
	}

	if err := h.service.DeleteWork(c.Context(), userID, userRole, workID); err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "You do not have permission to delete this work"})
		}
		log.Printf("Handler.DeleteWork: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to delete portfolio work"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ToggleWorkUpvote handles requests from authenticated users to upvote or downvote a work.
func (h *Handler) ToggleWorkUpvote(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	workID, err := strconv.ParseInt(c.Params("work_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid work ID"})
	}

	result, err := h.service.ToggleWorkUpvote(c.Context(), userID, workID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Portfolio work not found"})
		}
		log.Printf("Handler.ToggleWorkUpvote: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to upvote"})
	}
	return c.Status(fiber.StatusOK).JSON(result)
}
