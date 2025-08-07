package notification

import (
	"errors"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/pkg/utils"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handler handles HTTP requests for the notification module.
type Handler struct {
	service ServiceInterface
}

// NewHandler creates a new notification handler.
func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

// GetNotifications handles requests from an authenticated user to get their notifications.
func (h *Handler) GetNotifications(c echo.Context) error {
	// This is a protected route, so a user ID must exist in the context.
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	// Get pagination parameters from the request query string.
	page, limit := utils.GetPageLimit(c)

	notifications, total, err := h.service.GetNotificationsForUser(c.Request().Context(), userID, page, limit)
	if err != nil {
		c.Logger().Error("Handler.GetNotifications: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve notifications"})
	}

	// Return the data in a standardized paginated response format.
	return c.JSON(http.StatusOK, models.NewPaginatedResponse(notifications, page, limit, total))
}

// GetUnreadNotificationCount handles requests to get the count of a user's unread notifications.
func (h *Handler) GetUnreadNotificationCount(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	count, err := h.service.GetUnreadNotificationCount(c.Request().Context(), userID)
	if err != nil {
		c.Logger().Error("Handler.GetUnreadNotificationCount: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve unread notification count"})
	}

	// Return the count in a simple JSON object.
	return c.JSON(http.StatusOK, map[string]int64{"count": count})
}

// MarkAsRead handles requests to mark a single notification as read.
func (h *Handler) MarkAsRead(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	// Parse the notification ID from the URL path parameter.
	notificationID, err := strconv.ParseInt(c.Param("notification_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid notification ID"})
	}

	err = h.service.MarkNotificationAsRead(c.Request().Context(), notificationID, userID)
	if err != nil {
		// Check if the error is a "not found" error, which could mean the notification
		// doesn't exist or doesn't belong to the user.
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Notification not found"})
		}
		c.Logger().Error("Handler.MarkAsRead: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to mark notification as read"})
	}

	// A 204 No Content response is appropriate for a successful action that doesn't return a body.
	return c.NoContent(http.StatusNoContent)
}

// MarkAllAsRead handles requests to mark all of a user's notifications as read.
func (h *Handler) MarkAllAsRead(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}

	err = h.service.MarkAllNotificationsAsRead(c.Request().Context(), userID)
	if err != nil {
		c.Logger().Error("Handler.MarkAllAsRead: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to mark all notifications as read"})
	}

	return c.NoContent(http.StatusNoContent)
}
