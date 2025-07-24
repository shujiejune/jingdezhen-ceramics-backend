package course

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/internal/modules/user"
	"math"
	"time"
)

type ServiceInterface interface {
	GetAllCourses(ctx context.Context) ([]models.Course, error)
	GetQuizDetails(ctx context.Context, quizID int64) (*models.QuizContent, error)
	GetCourseDetails(ctx context.Context, userID string, courseID int64) (*models.Course, error)
	GetChapterContent(ctx context.Context, userID string, chapterID int64) (*models.CourseChapter, error)
	EnrollUserInCourse(ctx context.Context, userID string, courseID int64) error
	UpdateUserProgress(ctx context.Context, userID string, chapterID int64, progress models.UpdateProgressRequest) error
	AddNoteToChapter(ctx context.Context, userID string, chapterID int64, data models.AddNoteToEntityRequest) (*models.UserNote, error)
	SubmitAssignment(ctx context.Context, userID string, chapterID int64, assignmentID int64, data models.SubmitAssignmentRequest) (any, error)
	SubmitQuiz(ctx context.Context, userID string, chapterID int64, quizID int64, data models.SubmitQuizRequest) (any, error)
}

type Service struct {
	repo    RepositoryInterface
	userSvc user.ServiceInterface
}

func NewService(repo RepositoryInterface, userSvc user.ServiceInterface) ServiceInterface {
	return &Service{repo: repo, userSvc: userSvc}
}

func (s *Service) GetAllCourses(ctx context.Context) ([]models.Course, error) {
	return s.repo.FindAllCourses(ctx)
}

func (s *Service) GetQuizDetails(ctx context.Context, quizID int64) (*models.QuizContent, error) {
	quiz, err := s.repo.FindQuizWithAnswersByID(ctx, quizID)
	if err != nil {
		return nil, fmt.Errorf("service.GetQuizDetails: %w", err)
	}
	return quiz, nil
}

func (s *Service) GetCourseDetails(ctx context.Context, userID string, courseID int64) (*models.Course, error) {
	course, err := s.repo.FindCourseByID(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("service.GetCourseDetails.FindCourse: %w", err)
	}

	chapters, err := s.repo.FindChaptersByCourseID(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("service.GetCourseDetails.FindChapters: %w", err)
	}

	// If user is logged in, fetch their progress for these chapters
	if userID != "" && len(chapters) > 0 {
		chapterIDs := make([]int64, len(chapters))
		for i, ch := range chapters {
			chapterIDs[i] = ch.ID
		}
		progressMap, err := s.repo.GetUserProgressForChapters(ctx, userID, chapterIDs)
		if err != nil {
			// Log this error but don't fail the entire request
			fmt.Printf("WARN: could not get user progress for course %d: %v\n", courseID, err)
		}
		for i := range chapters {
			if progress, ok := progressMap[chapters[i].ID]; ok {
				chapters[i].ProgressPercentage = progress.ProgressPercentage
			}
		}
	}

	course.Chapters = chapters
	return course, nil
}

func (s *Service) GetChapterContent(ctx context.Context, userID string, chapterID int64) (*models.CourseChapter, error) {
	chapter, err := s.repo.FindChapterByID(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("service.GetChapterContent.FindChapter: %w", err)
	}

	isEnrolled := false
	if userID != "" {
		isEnrolled, err = s.repo.CheckUserEnrollment(ctx, userID, chapter.CourseID)
		if err != nil {
			return nil, fmt.Errorf("service.GetChapterContent.CheckEnrollment: %w", err)
		}
	}

	// Business Logic: Hide full content for non-enrolled users on protected chapters
	if !isEnrolled && !chapter.AvailableForGuests {
		chapter.ResourceLinks = nil
		chapter.ContentBlocks = nil // Only show the preview
	}

	// Get user progress for this specific chapter if logged in
	if userID != "" {
		progressMap, _ := s.repo.GetUserProgressForChapters(ctx, userID, []int64{chapterID})
		if progress, ok := progressMap[chapterID]; ok {
			chapter.ProgressPercentage = progress.ProgressPercentage
		}
	}

	return chapter, nil
}

func (s *Service) EnrollUserInCourse(ctx context.Context, userID string, courseID int64) error {
	// Business logic: check if course exists first
	_, err := s.repo.FindCourseByID(ctx, courseID)
	if err != nil {
		return fmt.Errorf("cannot enroll in non-existent course: %w", err)
	}
	return s.repo.EnrollUserInCourse(ctx, userID, courseID)
}

func (s *Service) UpdateUserProgress(ctx context.Context, userID string, chapterID int64, progress models.UpdateProgressRequest) error {
	// Business logic: check if user is enrolled in the course this chapter belongs to
	chapter, err := s.repo.FindChapterByID(ctx, chapterID)
	if err != nil {
		return err
	}
	isEnrolled, err := s.repo.CheckUserEnrollment(ctx, userID, chapter.CourseID)
	if err != nil {
		return err
	}
	if !isEnrolled {
		return models.ErrForbidden // User must be enrolled to update progress
	}
	return s.repo.UpdateUserProgress(ctx, userID, chapterID, progress)
}

func (s *Service) AddNoteToChapter(ctx context.Context, userID string, chapterID int64, data models.AddNoteToEntityRequest) (*models.UserNote, error) {
	// Business logic: check if chapter exists
	_, err := s.repo.FindChapterByID(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("cannot add note to non-existent chapter: %w", err)
	}

	entityType := "course_chapter"
	entityID := int(chapterID)

	noteData := models.CreateUserNoteData{
		Title:      data.Title,
		Content:    data.Content,
		EntityType: &entityType,
		EntityID:   &entityID,
	}
	return s.userSvc.CreateUserNote(ctx, userID, noteData)
}

func (s *Service) SubmitAssignment(ctx context.Context, userID string, chapterID int64, assignmentID int64, data models.SubmitAssignmentRequest) (any, error) {
	// Business logic:
	// 1. Check if within deadline.
	asgn, err := s.repo.FindAssignmentByID(ctx, assignmentID)
	if asgn.ApplyDeadline && asgn.Deadline != nil && time.Now().After(*asgn.Deadline) {
		return nil, models.ErrMissedDeadline
	}

	// 2. Check if user is enrolled.
	chapter, err := s.repo.FindChapterByID(ctx, chapterID)
	if err != nil {
		return nil, err
	}
	isEnrolled, err := s.repo.CheckUserEnrollment(ctx, userID, chapter.CourseID)
	if err != nil {
		return nil, err
	}
	if !isEnrolled {
		return nil, models.ErrForbidden // User must be enrolled to submit answers
	}

	// 3. Submit assgnment
	return s.repo.SubmitAssignment(ctx, userID, assignmentID, data.Answers)
}

func (s *Service) SubmitQuiz(ctx context.Context, userID string, chapterID int64, quizID int64, data models.SubmitQuizRequest) (any, error) {
	// Business logic:
	// 1. Check if user is enrolled.
	chapter, err := s.repo.FindChapterByID(ctx, chapterID)
	if err != nil {
		return nil, err
	}
	isEnrolled, err := s.repo.CheckUserEnrollment(ctx, userID, chapter.CourseID)
	if err != nil {
		return nil, err
	}
	if !isEnrolled {
		return nil, models.ErrForbidden // User must be enrolled to submit answers
	}

	// 2. Fetch the quiz questions and correct answers from the DB (e.g., from chapter_quizzes table).
	quiz, err := s.repo.FindQuizWithAnswersByID(ctx, quizID)
	if err != nil {
		return nil, fmt.Errorf("could not find quiz: %w", err)
	}

	// 3. Compare user's answers with correct answers.
	score := 0
	totalPoints := 0
	hasEssay := false
	correctAnswersForResponse := make(map[string]any)

	for _, question := range quiz.Questions {
		totalPoints += question.Points
		userAnswer, ok := data.Answers[question.ID]
		if !ok {
			continue // User didn't answer this question
		}

		// Store the correct answer to show the user later
		correctAnswersForResponse[question.ID] = question.Answer

		switch question.Type {
		case "single_choice":
			if correctAnswer, ok := question.Answer.(string); ok {
				if userAnswerStr, ok := userAnswer.(string); ok && correctAnswer == userAnswerStr {
					score += question.Points
				}
			}
		case "multiple_choice":
			correctAnswerSlice, ok1 := question.Answer.([]any)
			userAnswerSlice, ok2 := userAnswer.([]any)
			if !ok1 || !ok2 {
				// If the types are wrong, something is malformed. Skip scoring.
				continue
			}

			// Use maps for efficient lookup to find matches, regardless of order.
			correctSet := make(map[string]bool)
			for _, item := range correctAnswerSlice {
				if str, ok := item.(string); ok {
					correctSet[str] = true
				}
			}

			matchedCount := 0
			for _, item := range userAnswerSlice {
				if str, ok := item.(string); ok {
					if correctSet[str] {
						matchedCount++
					}
				}
			}

			if len(correctSet) > 0 {
				// Use floating point for division, then round to nearest integer.
				partialScore := (float64(matchedCount) / float64(len(correctSet))) * float64(question.Points)
				score += int(math.Round(partialScore))
			}
		case "true_or_false":
			if correctAnswer, ok := question.Answer; ok {
				if correctAnswer == userAnswer {
					score += question.Points
				}
			}
		case "fill_blanks":
			correctAnswerSlice, ok1 := question.Answer.([]any)
			userAnswerSlice, ok2 := userAnswer.([]any)
			if !ok1 || !ok2 {
				// If the types are wrong, something is malformed. Skip scoring.
				continue
			}

			matchedCount := 0
			for idx, item := range userAnswerSlice {
				if str, ok := item.(string); ok {
					if str == correctAnswerSlice[idx] {
						matchedCount++
					}
				}
			}

			partialScore := (float64(matchedCount) / float64(len(correctAnswerSlice))) * float64(question.Points)
			score += int(math.Round(partialScore))
		case "essay":
			hasEssay = true // Mark for manual grading
		}
	}

	// 4. Save the attempt and score to the DB (e.g., user_quiz_attempts table).
	attempt := models.QuizAttempt{
		UserID:      userID,
		QuizID:      quizID,
		Answers:     data.Answers, // Save what the user submitted
		Score:       score,
		SubmittedAt: time.Now(),
		Status:      "graded",
	}
	if hasEssay {
		attempt.Status = "pending_manual_grade"
	}

	savedAttempt, err := s.repo.SaveQuizAttempt(ctx, attempt)
	if err != nil {
		return nil, fmt.Errorf("failed to save quiz attempt: %w", err)
	}

	// 5. Return the results (e.g., score, correct answers).
	feedback := "Great job!"
	if score < totalPoints {
		feedback = "Good effort! Review the chapter content to improve your score."
	}
	if hasEssay {
		feedback += " Your essay questions are pending review."
	}

	result := &models.QuizAttemptResult{
		AttemptID:      savedAttempt.ID,
		Score:          score,
		TotalPoints:    totalPoints,
		Status:         attempt.Status,
		Feedback:       feedback,
		CorrectAnswers: correctAnswersForResponse,
	}

	return result, nil
}
