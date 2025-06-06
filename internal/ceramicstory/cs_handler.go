package ceramicstory

import (
	"jingdezhen-ceramics-backend/internal/models"
	"net/http"

	// "github.com/go-playground/validator/v10" // If you add admin routes for C/U/D
	"github.com/labstack/echo/v4"
)

// Handler handles HTTP requests for ceramic stories.
type Handler struct {
	service ServiceInterface
	// validate *validator.Validate // Needed for admin C/U/D operations
}

// NewHandler creates a new ceramic story handler.
func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
		// validate: validator.New(),
	}
}

// GetAllDynasties handles the request to get all ceramic stories.
// Corresponds to: csGroup.GET("", csHandler.GetAllDynasties)
func (h *Handler) GetAllDynasties(c echo.Context) error {
	ctx := c.Request().Context()
	stories, err := h.service.GetAllCeramicStories(ctx)
	if err != nil {
		// In a real app, you'd check the error type for more specific responses
		c.Logger().Error("Handler.GetAllDynasties: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve ceramic stories"})
	}

	if len(stories) == 0 {
		return c.JSON(http.StatusOK, []models.CeramicStory{}) // Return empty list, not an error
	}

	return c.JSON(http.StatusOK, stories)
}

// GetDynastyDetail handles the request to get details for a specific ceramic story.
// Corresponds to: csGroup.GET("/:dynasty_id_or_slug", csHandler.GetDynastyDetail)
func (h *Handler) GetDynastyDetail(c echo.Context) error {
	idOrSlug := c.Param("dynasty_id_or_slug")
	if idOrSlug == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Dynasty ID or slug parameter is required"})
	}

	ctx := c.Request().Context()
	story, err := h.service.GetCeramicStoryDetail(ctx, idOrSlug)
	if err != nil {
		if err == models.ErrNotFound || strings.Contains(err.Error(), models.ErrNotFound.Error()) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Ceramic story not found"})
		}
		c.Logger().Error("Handler.GetDynastyDetail: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve ceramic story details"})
	}

	return c.JSON(http.StatusOK, story)
}

// --- Admin Handlers (Example - uncomment and complete if you add admin routes) ---
/*
func (h *Handler) CreateCeramicStory(c echo.Context) error {
	// This route would need admin authentication middleware
	var req models.CreateCeramicStoryData
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body: " + err.Error()})
	}
	if err := h.validate.StructCtx(c.Request().Context(), req); err != nil { // Use StructCtx for context-aware validation
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	story, err := h.service.CreateCeramicStory(c.Request().Context(), req)
	if err != nil {
		// Check for specific errors like models.ErrConflict (slug taken)
		// if errors.Is(err, models.ErrConflict) {
		// 	  return c.JSON(http.StatusConflict, models.ErrorResponse{Message: "Slug already exists"})
		// }
		c.Logger().Error("Handler.CreateCeramicStory: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to create ceramic story"})
	}
	return c.JSON(http.StatusCreated, story)
}

func (h *Handler) UpdateCeramicStory(c echo.Context) error {
	// This route would need admin authentication middleware
	idStr := c.Param("id") // Assuming route is /admin/ceramicstory/:id
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid ID parameter"})
	}

	var req models.UpdateCeramicStoryData
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body: " + err.Error()})
	}
	if err := h.validate.StructCtx(c.Request().Context(), req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	story, err := h.service.UpdateCeramicStory(c.Request().Context(), id, req)
	if err != nil {
		// if errors.Is(err, models.ErrNotFound) {
		// 	 return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Ceramic story not found"})
		// }
		// if errors.Is(err, models.ErrConflict) {
		// 	 return c.JSON(http.StatusConflict, models.ErrorResponse{Message: "Slug already exists for another story"})
		// }
		c.Logger().Error("Handler.UpdateCeramicStory: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to update ceramic story"})
	}
	return c.JSON(http.StatusOK, story)
}

func (h *Handler) DeleteCeramicStory(c echo.Context) error {
	// This route would need admin authentication middleware
	idStr := c.Param("id") // Assuming route is /admin/ceramicstory/:id
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid ID parameter"})
	}

	err = h.service.DeleteCeramicStory(c.Request().Context(), id)
	if err != nil {
		// if errors.Is(err, models.ErrNotFound) {
		// 	 return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Ceramic story not found"})
		// }
		c.Logger().Error("Handler.DeleteCeramicStory: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to delete ceramic story"})
	}
	return c.NoContent(http.StatusNoContent)
}
*/
