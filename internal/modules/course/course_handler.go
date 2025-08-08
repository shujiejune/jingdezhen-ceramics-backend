package course

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

func (h *Handler) GetAllCourses(c *fiber.Ctx) error {
	courses, err := h.service.GetAllCourses(c.Context())
	if err != nil {
		log.Printf("Handler.GetAllCourses: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve courses"})
	}
	return c.Status(fiber.StatusOK).JSON(courses)
}

func (h *Handler) GetCourseDetails(c *fiber.Ctx) error {
	userID, _ := utils.GetUserIDFromContext(c) // OK if this fails for guests
	courseID, err := strconv.ParseInt(c.Params("course_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid course ID"})
	}

	course, err := h.service.GetCourseDetails(c.Context(), userID, courseID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Course not found"})
		}
		log.Printf("Handler.GetCourseDetails: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve course details"})
	}
	return c.Status(fiber.StatusOK).JSON(course)
}

func (h *Handler) GetChapterContent(c *fiber.Ctx) error {
	userID, _ := utils.GetUserIDFromContext(c) // OK if this fails for guests
	chapterID, err := strconv.ParseInt(c.Params("chapter_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid chapter ID"})
	}

	// This single endpoint handles both public preview and full content for enrolled users.
	// The service layer contains the logic to decide what to return.
	chapter, err := h.service.GetChapterContent(c.Context(), userID, chapterID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Chapter not found"})
		}
		log.Printf("Handler.GetChapterContent: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve chapter content"})
	}
	return c.Status(fiber.StatusOK).JSON(chapter)
}

// --- Protected Handlers ---

func (h *Handler) EnrollCourse(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	courseID, err := strconv.ParseInt(c.Params("course_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid course ID"})
	}

	if err := h.service.EnrollUserInCourse(c.Context(), userID, courseID); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Course not found"})
		}
		log.Printf("Handler.EnrollCourse: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to enroll in course"})
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) GetEnrolledCourses(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}

	courses, err := h.service.GetEnrolledCourses(c.Context(), userID)
	if err != nil {
		log.Printf("Handler.GetEnrolledCourses: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve enrolled courses"})
	}

	// Return an empty list if the user is not enrolled in any courses, not an error.
	if len(courses) == 0 {
		return c.Status(fiber.StatusOK).JSON([]models.EnrolledCourseResponse{})
	}

	return c.Status(fiber.StatusOK).JSON(courses)
}

// GetFullChapterContentForEnrolled is an explicit endpoint for enrolled users.
// It's functionally similar to GetChapterContent, but its presence in the protected group
// makes the API contract clearer. The service logic ensures GetChapterContent is also secure.
func (h *Handler) GetFullChapterContentForEnrolled(c *fiber.Ctx) error {
	return h.GetChapterContent(c)
}

// MarkContentBlockComplete handles requests to mark a passive content block (video, reading) as complete.
// Corresponds to: POST /courses/:course_id/chapters/:chapter_id/blocks/:block_id/complete
func (h *Handler) MarkContentBlockComplete(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	courseID, err := strconv.ParseInt(c.Params("course_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid course ID"})
	}
	chapterID, err := strconv.ParseInt(c.Params("chapter_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid chapter ID"})
	}
	blockID, err := strconv.ParseInt(c.Params("block_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid content block ID"})
	}

	// This request has no body. The action is in the URL.
	if err := h.service.MarkContentBlockComplete(c.Context(), userID, courseID, chapterID, blockID); err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "You must be enrolled to mark progress"})
		}
		if errors.Is(err, models.ErrInvalidOperation) {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: err.Error()})
		}
		log.Printf("Handler.MarkContentBlockComplete: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to update progress"})
	}

	return c.SendStatus(fiber.StatusOK)
}

// UpdateVideoProgress handles requests to save the last stopped-at time for a video.
// Corresponds to: POST /courses/:course_id/chapters/:chapter_id/blocks/:block_id/video-progress
func (h *Handler) UpdateVideoProgress(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	blockID, err := strconv.ParseInt(c.Params("block_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid content block ID"})
	}

	var req models.UpdateVideoProgressRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	if err := h.service.UpdateVideoProgress(c.Context(), userID, blockID, req); err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "You must be enrolled to update video progress"})
		}
		if errors.Is(err, models.ErrInvalidOperation) {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: err.Error()})
		}
		log.Printf("Handler.UpdateVideoProgress: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to update video progress"})
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) AddNoteToChapter(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	chapterID, err := strconv.ParseInt(c.Params("chapter_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid chapter ID"})
	}

	var req models.AddNoteToEntityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	note, err := h.service.AddNoteToChapter(c.Context(), userID, chapterID, req)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Cannot add note: chapter not found"})
		}
		log.Printf("Handler.AddNoteToChapter: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to add note"})
	}
	return c.Status(fiber.StatusCreated).JSON(note)
}

func (h *Handler) SubmitAssignment(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	chapterID, err := strconv.ParseInt(c.Params("chapter_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid chapter ID"})
	}
	assignmentID, err := strconv.ParseInt(c.Params("assignment_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid assignment ID"})
	}

	var req models.SubmitAssignmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	result, err := h.service.SubmitAssignment(c.Context(), userID, chapterID, assignmentID, req)
	if err != nil {
		// Handle specific errors like assignment not found, user not enrolled, etc.
		log.Printf("Handler.SubmitAssignment: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to submit assignment"})
	}
	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *Handler) GetQuiz(c *fiber.Ctx) error {
	quizID, err := strconv.ParseInt(c.Params("quiz_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid quiz ID"})
	}

	quiz, err := h.service.GetQuizDetails(c.Context(), quizID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{Message: "Quiz not found"})
		}
		log.Printf("Handler.GetQuiz: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to retrieve quiz details"})
	}
	return c.Status(fiber.StatusOK).JSON(quiz)
}

func (h *Handler) SubmitQuiz(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: err.Error()})
	}
	chapterID, err := strconv.ParseInt(c.Params("chapter_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid chapter ID"})
	}
	quizID, err := strconv.ParseInt(c.Params("quiz_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid quiz ID"})
	}

	var req models.SubmitQuizRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Invalid request body"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{Message: "Validation failed: " + err.Error()})
	}

	result, err := h.service.SubmitQuiz(c.Context(), userID, chapterID, quizID, req)
	if err != nil {
		// Handle specific errors like quiz not found, user not enrolled, etc.
		log.Printf("Handler.SubmitQuiz: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{Message: "Failed to submit quiz"})
	}
	return c.Status(fiber.StatusOK).JSON(result)
}
