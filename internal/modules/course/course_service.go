package course

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/internal/modules/note"
	"log"
	"math"
	"strings"
	"time"
)

type ServiceInterface interface {
	GetAllCourses(ctx context.Context) ([]models.Course, error)
	GetQuizDetails(ctx context.Context, quizID int64) (*models.Quiz, error)
	GetCourseDetails(ctx context.Context, userID string, courseID int64) (*models.Course, error)
	GetChapterContent(ctx context.Context, userID string, chapterID int64) (*models.CourseChapter, error)
	EnrollUserInCourse(ctx context.Context, userID string, courseID int64) error
	GetEnrolledCourses(ctx context.Context, userID string) ([]models.EnrolledCourseResponse, error)
	MarkContentBlockComplete(ctx context.Context, userID string, courseID, chapterID, blockID int64) error
	UpdateVideoProgress(ctx context.Context, userID string, blockID int64, data models.UpdateVideoProgressRequest) error
	AddNoteToChapter(ctx context.Context, userID string, chapterID int64, data models.AddNoteToEntityRequest) (*models.UserNote, error)
	SubmitAssignment(ctx context.Context, userID string, chapterID int64, assignmentID int64, data models.SubmitAssignmentRequest) (any, error)
	SubmitQuiz(ctx context.Context, userID string, chapterID int64, quizID int64, data models.SubmitQuizRequest) (any, error)
}

type Service struct {
	repo    RepositoryInterface
	noteSvc note.ServiceInterface
}

func NewService(repo RepositoryInterface, noteSvc note.ServiceInterface) ServiceInterface {
	return &Service{repo: repo, noteSvc: noteSvc}
}

func (s *Service) GetAllCourses(ctx context.Context) ([]models.Course, error) {
	return s.repo.FindAllCourses(ctx)
}

func (s *Service) GetQuizDetails(ctx context.Context, quizID int64) (*models.Quiz, error) {
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

	isEnrolled := false
	if userID != "" {
		isEnrolled, err = s.repo.CheckUserEnrollment(ctx, userID, courseID)
		if err != nil {
			return nil, fmt.Errorf("service.GetCourseDetails.CheckEnrollment: %w", err)
		}
	}

	if isEnrolled {
		go s.repo.UpdateLastVisitedAt(context.Background(), userID, courseID)
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

	// Hide full content for non-enrolled users on protected chapters
	if !isEnrolled && !chapter.AvailableForGuests {
		chapter.ResourceLinks = nil
		chapter.ContentBlocks = nil
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

func (s *Service) GetEnrolledCourses(ctx context.Context, userID string) ([]models.EnrolledCourseResponse, error) {
	courses, err := s.repo.FindEnrolledCoursesByUserID(ctx, userID)
	if err != nil {
		// No complex business logic here for now, just pass the result through.
		// This layer exists to add such logic later if needed.
		return nil, fmt.Errorf("service.GetEnrolledCourses: %w", err)
	}
	return courses, nil
}

// MarkContentBlockComplete handles the logic for when a user marks a content block as done.
func (s *Service) MarkContentBlockComplete(ctx context.Context, userID string, courseID, chapterID, blockID int64) error {
	// 1. Check if the user is actually enrolled in the course.
	isEnrolled, err := s.repo.CheckUserEnrollment(ctx, userID, courseID)
	if err != nil {
		return fmt.Errorf("service.MarkContentBlockComplete.CheckEnrollment: %w", err)
	}
	if !isEnrolled {
		return models.ErrForbidden // User must be enrolled to mark progress.
	}

	// 2. Check if the block belongs to the chapter.
	// This prevents a user from sending a valid blockID with a mismatched chapterID.
	block, err := s.repo.FindContentBlockByID(ctx, blockID)
	if err != nil {
		return err
	}
	if block.ChapterID != chapterID || block.CourseID != courseID {
		return models.ErrForbidden
	}

	if block.Type != "video" && block.Type != "reading" {
		return fmt.Errorf("%w: cannot mark a %s block as complete via this endpoint", models.ErrInvalidOperation, block.Type)
	}

	// 3. Call the repository to save the completion record.
	err = s.repo.SavePassiveBlockCompletion(ctx, userID, blockID)
	if err != nil {
		return fmt.Errorf("service.MarkContentBlockComplete.SaveCompletion: %w", err)
	}

	// 4. After a block is completed, recalculate the overall chapter progress percentage.
	// This is a crucial step to keep the data in sync.
	go func() {
		bgCtx := context.Background()
		if err := s.repo.CalculateChapterProgress(bgCtx, userID, chapterID); err != nil {
			log.Printf("ERROR: Failed to recalculate chapter progress for user %s, chapter %d: %v", userID, chapterID, err)
		}
	}()

	return nil
}

// UpdateVideoProgress handles saving the last stopped-at time for a video.
func (s *Service) UpdateVideoProgress(ctx context.Context, userID string, blockID int64, data models.UpdateVideoProgressRequest) error {
	// Check if the user is enrolled in the course this block belongs to.
	// This requires fetching the block to find its chapter, then its course.
	block, err := s.repo.FindContentBlockByID(ctx, blockID)
	if err != nil {
		return fmt.Errorf("service.UpdateVideoProgress.FindBlock: %w", err)
	}
	if block.Type != "video" {
		return fmt.Errorf("%w: cannot update video progress on a non-video block of type %s", models.ErrInvalidOperation, block.Type)
	}

	isEnrolled, err := s.repo.CheckUserEnrollment(ctx, userID, block.CourseID)
	if err != nil {
		return fmt.Errorf("service.UpdateVideoProgress.CheckEnrollment: %w", err)
	}
	if !isEnrolled {
		return models.ErrForbidden
	}

	// Call the repository to save the video progress.
	err = s.repo.SaveVideoProgress(ctx, userID, blockID, data.LastStoppedAt)
	if err != nil {
		return fmt.Errorf("service.UpdateVideoProgress.SaveProgress: %w", err)
	}

	// After saving, check if the video is now complete.
	if videoContent, ok := block.Content.(models.VideoContent); ok {
		// Add a small buffer (e.g., 5 seconds) to account for player inaccuracies.
		if videoContent.Duration > 0 && data.LastStoppedAt >= videoContent.Duration-5 {
			log.Printf("INFO: Video block %d completed for user %s. Auto-marking as complete.\n", blockID, userID)
			// Call the MarkContentBlockComplete logic.
			go func() {
				bgCtx := context.Background()
				if err := s.MarkContentBlockComplete(bgCtx, userID, block.CourseID, block.ChapterID, blockID); err != nil {
					log.Printf("WARN: Failed to auto-mark video as complete after finishing: %v", err)
				}
			}()
		}
	}

	return nil
}

func (s *Service) AddNoteToChapter(ctx context.Context, userID string, chapterID int64, data models.AddNoteToEntityRequest) (*models.UserNote, error) {
	// Business logic: check if chapter exists
	_, err := s.repo.FindChapterByID(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("cannot add note to non-existent chapter: %w", err)
	}

	noteData := models.CreateUserNoteData{
		Title:      data.Title,
		Content:    data.Content,
		EntityType: &data.EntityType,
		EntityID:   data.EntityID,
	}
	return s.noteSvc.CreateUserNote(ctx, userID, noteData)
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
	submission := models.AssignmentSubmission{
		AssignmentID: assignmentID,
		UserID:       userID,
		Answers:      data.Answers,
		Status:       "ungraded",
		SubmittedAt:  time.Now(),
	}
	result, err := s.repo.SubmitAssignment(ctx, submission)
	if err != nil {
		return nil, err
	}

	// 4. After submission, trigger progress recalculation
	go func() {
		bgCtx := context.Background()
		if err := s.repo.CalculateChapterProgress(bgCtx, userID, chapterID); err != nil {
			log.Printf("ERROR: Failed to recalculate chapter progress after assignment submission for user %s, chapter %d: %v", userID, chapterID, err)
		}
	}()

	return result, nil
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
			correctAnswer, ok1 := question.Answer.(bool)
			userAnswerBool, ok2 := userAnswer.(bool)
			if ok1 && ok2 {
				if correctAnswer == userAnswerBool {
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

			// Prevent division by zero if the quiz question was misconfigured
			if len(correctAnswerSlice) == 0 {
				continue
			}

			matchedCount := 0
			for i := 0; i < len(correctAnswerSlice); i++ {
				if i >= len(userAnswerSlice) {
					break
				}
				userAnswerStr, okUser := userAnswerSlice[i].(string)
				correctAnswerStr, okCorrect := correctAnswerSlice[i].(string)
				if okUser && okCorrect {
					// Use strings.EqualFold for case-insensitive comparison
					if strings.EqualFold(userAnswerStr, correctAnswerStr) {
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

	// 5. After saving the attempt, trigger progress recalculation
	go func() {
		bgCtx := context.Background()
		if err := s.repo.CalculateChapterProgress(bgCtx, userID, chapterID); err != nil {
			log.Printf("ERROR: Failed to recalculate chapter progress after quiz submission for user %s, chapter %d: %v", userID, chapterID, err)
		}
	}()

	// 6. Return the results (e.g., score, correct answers).
	feedback := "Great job!"
	if score < totalPoints {
		feedback = "Good effort! Review the chapter content to improve your score."
	}
	if hasEssay {
		feedback += " Your essay is pending review."
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
