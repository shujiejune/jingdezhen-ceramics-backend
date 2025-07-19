package course

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

func (h *Handler) GetAllCourses(c echo.Context) error {
	courses, err := h.service.GetAllCourses(c.Request().Context())
	if err != nil {
		c.Logger().Error("Handler.GetAllCourses: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve courses"})
	}
	return c.JSON(http.StatusOK, courses)
}

func (h *Handler) GetCourseDetails(c echo.Context) error {
	userID, _ := utils.GetUserIDFromContext(c) // OK if this fails for guests
	courseID, err := strconv.ParseInt(c.Param("course_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid course ID"})
	}

	course, err := h.service.GetCourseDetails(c.Request().Context(), userID, courseID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Course not found"})
		}
		c.Logger().Error("Handler.GetCourseDetails: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve course details"})
	}
	return c.JSON(http.StatusOK, course)
}

func (h *Handler) GetChapterContent(c echo.Context) error {
	userID, _ := utils.GetUserIDFromContext(c) // OK if this fails for guests
	chapterID, err := strconv.ParseInt(c.Param("chapter_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid chapter ID"})
	}

	// This single endpoint handles both public preview and full content for enrolled users.
	// The service layer contains the logic to decide what to return.
	chapter, err := h.service.GetChapterContent(c.Request().Context(), userID, chapterID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Chapter not found"})
		}
		c.Logger().Error("Handler.GetChapterContent: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to retrieve chapter content"})
	}
	return c.JSON(http.StatusOK, chapter)
}

// --- Protected Handlers ---

func (h *Handler) EnrollCourse(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	courseID, err := strconv.ParseInt(c.Param("course_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid course ID"})
	}

	if err := h.service.EnrollUserInCourse(c.Request().Context(), userID, courseID); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Course not found"})
		}
		c.Logger().Error("Handler.EnrollCourse: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to enroll in course"})
	}

	return c.NoContent(http.StatusOK)
}

// GetFullChapterContentForEnrolled is an explicit endpoint for enrolled users.
// It's functionally similar to GetChapterContent, but its presence in the protected group
// makes the API contract clearer. The service logic ensures GetChapterContent is also secure.
func (h *Handler) GetFullChapterContentForEnrolled(c echo.Context) error {
	return h.GetChapterContent(c)
}

func (h *Handler) UpdateProgress(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	chapterID, err := strconv.ParseInt(c.Param("chapter_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid chapter ID"})
	}

	var req models.UpdateProgressRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	if err := h.service.UpdateUserProgress(c.Request().Context(), userID, chapterID, req); err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{Message: "You must be enrolled to update progress"})
		}
		c.Logger().Error("Handler.UpdateProgress: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to update progress"})
	}

	return c.NoContent(http.StatusOK)
}

func (h *Handler) AddNoteToChapter(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	chapterID, err := strconv.ParseInt(c.Param("chapter_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid chapter ID"})
	}

	var req models.AddNoteToArtworkRequest // Reusing this struct for simple Title/Content
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	note, err := h.service.AddNoteToChapter(c.Request().Context(), userID, chapterID, req)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{Message: "Cannot add note: chapter not found"})
		}
		c.Logger().Error("Handler.AddNoteToChapter: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to add note"})
	}
	return c.JSON(http.StatusCreated, note)
}

func (h *Handler) SubmitQuiz(c echo.Context) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: err.Error()})
	}
	chapterID, err := strconv.ParseInt(c.Param("chapter_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid chapter ID"})
	}
	quizID, err := strconv.ParseInt(c.Param("quiz_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid quiz ID"})
	}

	var req models.SubmitQuizRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	result, err := h.service.SubmitQuiz(c.Request().Context(), userID, chapterID, quizID, req)
	if err != nil {
		// Handle specific errors like quiz not found, user not enrolled, etc.
		c.Logger().Error("Handler.SubmitQuiz: ", err)
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "Failed to submit quiz"})
	}
	return c.JSON(http.StatusOK, result)
}
