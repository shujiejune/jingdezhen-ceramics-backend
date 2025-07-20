package portfolio

import (
	"errors"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/pkg/utils"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
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
func (h *Handler) GetWorks(c echo.Context) error {
	userID, _ := utils.GetUserIDFromContext(c) // Optional: for kudos status
	page, limit := utils.GetPageLimit(c)
	category := c.QueryParam("category")
	sort := c.QueryParam("sort")

	filters := models.PortfolioFilters{
		Page:     page,
		Limit:    limit,
		Category: category,
		Sort:     sort,
	}

	works, total, err := h.service.GetWorks(c.Request().Context(), userID, filters)
	if err != nil {
		c.Logger().Error("Handler.GetWorks: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve portfolio works"})
	}

	return c.JSON(http.StatusOK, models.NewPaginatedResponse(works, page, limit, total))
}

// GetWorkByID handles requests to get a single, detailed portfolio work.
func (h *Handler) GetWorkByID(c echo.Context) error {
	userID, _ := utils.GetUserIDFromContext(c)
	workID, err := strconv.ParseInt(c.Param("work_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid work ID"})
	}

	work, err := h.service.GetWorkByID(c.Request().Context(), userID, workID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Portfolio work not found"})
		}
		c.Logger().Error("Handler.GetWorkByID: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve portfolio work"})
	}

	return c.JSON(http.StatusOK, work)
}

// --- Protected Handlers ---

// CreateWork handles requests from authenticated users to create a new portfolio work.
func (h *Handler) CreateWork(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	var req models.CreateWorkRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	work, err := h.service.CreateWork(c.Request().Context(), userID, req)
	if err != nil {
		c.Logger().Error("Handler.CreateWork: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to create portfolio work"})
	}
	return c.JSON(http.StatusCreated, work)
}

// UpdateWork handles requests from authenticated users to update their own portfolio work.
func (h *Handler) UpdateWork(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	workID, err := strconv.ParseInt(c.Param("work_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid work ID"})
	}

	var req models.UpdateWorkRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	work, err := h.service.UpdateWork(c.Request().Context(), userID, workID, req)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{Message: "You do not have permission to edit this work"})
		}
		c.Logger().Error("Handler.UpdateWork: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to update portfolio work"})
	}
	return c.JSON(http.StatusOK, work)
}

// DeleteWork handles requests to delete a portfolio work, checking for ownership or admin role.
func (h *Handler) DeleteWork(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	userRole := c.Get("userRole").(string)

	workID, err := strconv.ParseInt(c.Param("work_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid work ID"})
	}

	if err := h.service.DeleteWork(c.Request().Context(), userID, userRole, workID); err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{Message: "You do not have permission to delete this work"})
		}
		c.Logger().Error("Handler.DeleteWork: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to delete portfolio work"})
	}
	return c.NoContent(http.StatusNoContent)
}

// LeaveKudo handles requests from authenticated users to leave a kudo on a work.
func (h *Handler) LeaveKudo(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	workID, err := strconv.ParseInt(c.Param("work_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid work ID"})
	}

	newCount, err := h.service.LeaveKudo(c.Request().Context(), userID, workID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Portfolio work not found"})
		}
		c.Logger().Error("Handler.LeaveKudo: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to leave kudo"})
	}
	return c.JSON(http.StatusOK, map[string]int{"kudos_count": newCount})
}
