package course

import (
	"context"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/internal/modules/user"
)

// PublicAccessChapterLimit defines how many chapters guests can view fully.
const PublicAccessChapterLimit = 2

type ServiceInterface interface {
	GetAllCourses(ctx context.Context) ([]models.Course, error)
	GetCourseDetails(ctx context.Context, userID string, courseID int64) (*models.Course, error)
	GetChapterContent(ctx context.Context, userID string, chapterID int64) (*models.CourseChapter, error)
	EnrollUserInCourse(ctx context.Context, userID string, courseID int64) error
	UpdateUserProgress(ctx context.Context, userID string, chapterID int64, progress models.UpdateProgressRequest) error
	AddNoteToChapter(ctx context.Context, userID string, chapterID int64, data models.AddNoteToArtworkRequest) (*models.UserNote, error)
	SubmitQuiz(ctx context.Context, userID string, chapterID int64, quizID int64, data models.SubmitQuizRequest) (interface{}, error)
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
	if !isEnrolled && chapter.DisplayOrder > PublicAccessChapterLimit {
		chapter.VideoURL = ""
		chapter.Content = "" // Only show the preview
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

func (s *Service) AddNoteToChapter(ctx context.Context, userID string, chapterID int64, data models.AddNoteToArtworkRequest) (*models.UserNote, error) {
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

func (s *Service) SubmitQuiz(ctx context.Context, userID string, chapterID int64, quizID int64, data models.SubmitQuizRequest) (interface{}, error) {
	// Business logic:
	// 1. Check if user is enrolled.
	// 2. Fetch the quiz questions and correct answers from the DB (e.g., from chapter_quizzes table).
	// 3. Compare user's answers with correct answers.
	// 4. Calculate score.
	// 5. Save the attempt and score to the DB (e.g., user_quiz_attempts table).
	// 6. Return the results (e.g., score, correct answers).
	return map[string]interface{}{"message": "Quiz submission successful", "score": "100%"}, nil // Placeholder
}
