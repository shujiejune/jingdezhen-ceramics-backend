package gallery

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

func (h *Handler) GetArtworks(c echo.Context) error {
	userID, _ := utils.GetUserIDFromContext(c) // It's ok if this fails for a guest
	page, limit := utils.GetPageLimit(c)
	category := c.QueryParam("category")
	artistID, _ := strconv.ParseInt(c.QueryParam("artist"), 10, 64)

	filters := models.ArtworkFilters{
		Category: category,
		ArtistID: artistID,
		Page:     page,
		Limit:    limit,
	}

	artworks, total, err := h.service.GetArtworks(c.Request().Context(), userID, filters)
	if err != nil {
		c.Logger().Error("Handler.GetArtworks: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve artworks"})
	}

	return c.JSON(http.StatusOK, models.NewPaginatedResponse(artworks, page, limit, total))
}

func (h *Handler) GetArtworkByID(c echo.Context) error {
	userID, _ := utils.GetUserIDFromContext(c)
	artworkID, err := strconv.ParseInt(c.Param("artwork_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid artwork ID"})
	}

	artwork, err := h.service.GetArtworkByID(c.Request().Context(), userID, artworkID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Artwork not found"})
		}
		c.Logger().Error("Handler.GetArtworkByID: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve artwork"})
	}

	return c.JSON(http.StatusOK, artwork)
}

func (h *Handler) GetArtists(c echo.Context) error {
	page, limit := utils.GetPageLimit(c)
	artists, total, err := h.service.GetArtists(c.Request().Context(), page, limit)
	if err != nil {
		c.Logger().Error("Handler.GetArtists: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve artists"})
	}
	return c.JSON(http.StatusOK, models.NewPaginatedResponse(artists, page, limit, total))
}

func (h *Handler) GetArtistByID(c echo.Context) error {
	artistID, err := strconv.ParseInt(c.Param("artist_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid artist ID"})
	}
	artist, err := h.service.GetArtistByID(c.Request().Context(), artistID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Artist not found"})
		}
		c.Logger().Error("Handler.GetArtistByID: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve artist"})
	}
	return c.JSON(http.StatusOK, artist)
}

func (h *Handler) GetGalleryCategories(c echo.Context) error {
	categories, err := h.service.GetGalleryCategories(c.Request().Context())
	if err != nil {
		c.Logger().Error("Handler.GetGalleryCategories: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve categories"})
	}
	return c.JSON(http.StatusOK, categories)
}

// --- Protected Handlers ---

func (h *Handler) MarkAsFavorite(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	artworkID, err := strconv.ParseInt(c.Param("artwork_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid artwork ID"})
	}

	if err := h.service.MarkAsFavorite(c.Request().Context(), userID, artworkID); err != nil {
		c.Logger().Error("Handler.MarkAsFavorite: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to mark as favorite"})
	}

	return c.NoContent(http.StatusOK) // Or http.StatusCreated
}

func (h *Handler) UnmarkAsFavorite(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	artworkID, err := strconv.ParseInt(c.Param("artwork_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid artwork ID"})
	}

	if err := h.service.UnmarkAsFavorite(c.Request().Context(), userID, artworkID); err != nil {
		c.Logger().Error("Handler.UnmarkAsFavorite: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to unmark as favorite"})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) AddNoteToArtwork(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	artworkID, err := strconv.ParseInt(c.Param("artwork_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid artwork ID"})
	}

	var req models.AddNoteToArtworkRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	note, err := h.service.AddNoteToArtwork(c.Request().Context(), userID, artworkID, req)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Cannot add note: artwork not found"})
		}
		c.Logger().Error("Handler.AddNoteToArtwork: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to add note"})
	}

	return c.JSON(http.StatusCreated, note)
}
