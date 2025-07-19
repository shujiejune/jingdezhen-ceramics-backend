package engage

import (
	"errors"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/pkg/utils"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	service ServiceInterface
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{service: service}
}

// GetActivities handles the request to get a paginated list of activities.
func (h *Handler) GetActivities(c echo.Context) error {
	page, limit := utils.GetPageLimit(c)
	ctx := c.Request().Context()

	activities, total, err := h.service.GetActivities(ctx, page, limit)
	if err != nil {
		c.Logger().Error("Handler.GetActivities: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve activities"})
	}

	return c.JSON(http.StatusOK, models.NewPaginatedResponse(activities, page, limit, total))
}

// GetActivityArticle handles the request to get a detailed article.
func (h *Handler) GetActivityArticle(c echo.Context) error {
	idOrSlug := c.Param("activity_id_or_slug")
	if idOrSlug == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Activity ID or slug parameter is required"})
	}

	ctx := c.Request().Context()
	article, err := h.service.GetActivityArticle(ctx, idOrSlug)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) || strings.Contains(err.Error(), models.ErrNotFound.Error()) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Article not found"})
		}
		c.Logger().Error("Handler.GetActivityArticle: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve article"})
	}

	return c.JSON(http.StatusOK, article)
}
