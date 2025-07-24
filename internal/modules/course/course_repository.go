package course

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBExecutor interface for query execution
type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type RepositoryInterface interface {
	FindAllCourses(ctx context.Context) ([]models.Course, error)
	FindCourseByID(ctx context.Context, courseID int64) (*models.Course, error)
	FindChaptersByCourseID(ctx context.Context, courseID int64) ([]models.CourseChapter, error)
	FindChapterByID(ctx context.Context, chapterID int64) (*models.CourseChapter, error)
	FindQuizWithAnswersByID(ctx context.Context, quizID int64) (*models.QuizContent, error)
	CheckUserEnrollment(ctx context.Context, userID string, courseID int64) (bool, error)
	EnrollUserInCourse(ctx context.Context, userID string, courseID int64) error
	GetUserProgressForChapters(ctx context.Context, userID string, chapterIDs []int64) (map[int64]models.UserChapterProgress, error)
	UpdateUserProgress(ctx context.Context, userID string, chapterID int64, progress models.UpdateProgressRequest) error
	SaveQuizAttempt(ctx context.Context, attempt models.QuizAttempt) (*models.QuizAttempt, error)
}

type Repository struct {
	db       *pgxpool.Pool
	executor DBExecutor
}

func NewRepository(db *pgxpool.Pool) RepositoryInterface {
	return &Repository{db: db, executor: db}
}

func (r *Repository) FindAllCourses(ctx context.Context) ([]models.Course, error) {
	courses := []models.Course{}
	query := `SELECT id, title, description, thumbnail_url, created_at, updated_at FROM courses ORDER BY created_at ASC`
	rows, err := r.executor.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository.FindAllCourses.Query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var course models.Course
		if err := rows.Scan(&course.ID, &course.Title, &course.Description, &course.ThumbnailURL, &course.CreatedAt, &course.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository.FindAllCourses.Scan: %w", err)
		}
		courses = append(courses, course)
	}
	return courses, nil
}

func (r *Repository) FindCourseByID(ctx context.Context, courseID int64) (*models.Course, error) {
	var course models.Course
	query := `SELECT id, title, description, thumbnail_url, created_at, updated_at FROM courses WHERE id = $1`
	err := r.executor.QueryRow(ctx, query, courseID).Scan(&course.ID, &course.Title, &course.Description, &course.ThumbnailURL, &course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindCourseByID: %w", err)
	}
	return &course, nil
}

func (r *Repository) FindChaptersByCourseID(ctx context.Context, courseID int64) ([]models.CourseChapter, error) {
	chapters := []models.CourseChapter{}
	// Note: We don't select full content here for efficiency. Full content is fetched on demand.
	query := `SELECT id, course_id, title, display_order, content_preview FROM course_chapters WHERE course_id = $1 ORDER BY display_order ASC`
	rows, err := r.executor.Query(ctx, query, courseID)
	if err != nil {
		return nil, fmt.Errorf("repository.FindChaptersByCourseID.Query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var chapter models.CourseChapter
		if err := rows.Scan(&chapter.ID, &chapter.CourseID, &chapter.Title, &chapter.DisplayOrder, &chapter.ContentPreview); err != nil {
			return nil, fmt.Errorf("repository.FindChaptersByCourseID.Scan: %w", err)
		}
		chapters = append(chapters, chapter)
	}
	return chapters, nil
}

func (r *Repository) FindChapterByID(ctx context.Context, chapterID int64) (*models.CourseChapter, error) {
	var chapter models.CourseChapter
	query := `SELECT id, course_id, title, display_order, video_url, content, content_preview FROM course_chapters WHERE id = $1`
	err := r.executor.QueryRow(ctx, query, chapterID).Scan(
		&chapter.ID, &chapter.CourseID, &chapter.Title, &chapter.DisplayOrder,
		&chapter.VideoURL, &chapter.Content, &chapter.ContentPreview,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindChapterByID: %w", err)
	}
	return &chapter, nil
}

// FindQuizWithAnswersByID retrieves the full quiz structure, including correct answers, from the database.
// This should only be called by the service layer for scoring, not sent directly to the client.
func (r *Repository) FindQuizWithAnswersByID(ctx context.Context, quizID int64) (*models.QuizContent, error) {
	var quiz models.QuizContent
	var questionsJSON []byte // We'll scan the JSONB data into a byte slice first

	query := `SELECT id, title, questions FROM quizzes WHERE id = $1`
	err := r.executor.QueryRow(ctx, query, quizID).Scan(&quiz.ID, &quiz.Title, &questionsJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindQuizWithAnswersByID: %w", err)
	}

	// Unmarshal the JSONB data into the Questions slice
	if err := json.Unmarshal(questionsJSON, &quiz.Questions); err != nil {
		return nil, fmt.Errorf("repository.FindQuizWithAnswersByID.Unmarshal: %w", err)
	}

	return &quiz, nil
}

func (r *Repository) CheckUserEnrollment(ctx context.Context, userID string, courseID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM user_enrollments WHERE user_id = $1 AND course_id = $2)`
	err := r.executor.QueryRow(ctx, query, userID, courseID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("repository.CheckUserEnrollment: %w", err)
	}
	return exists, nil
}

func (r *Repository) EnrollUserInCourse(ctx context.Context, userID string, courseID int64) error {
	query := `INSERT INTO user_enrollments (user_id, course_id) VALUES ($1, $2) ON CONFLICT (user_id, course_id) DO NOTHING`
	_, err := r.executor.Exec(ctx, query, userID, courseID)
	if err != nil {
		return fmt.Errorf("repository.EnrollUserInCourse: %w", err)
	}
	return nil
}

func (r *Repository) GetUserProgressForChapters(ctx context.Context, userID string, chapterIDs []int64) (map[int64]models.UserChapterProgress, error) {
	progressMap := make(map[int64]models.UserChapterProgress)
	if userID == "" || len(chapterIDs) == 0 {
		return progressMap, nil
	}

	query := `SELECT chapter_id, progress_percentage, video_last_stopped_at FROM user_chapter_progress WHERE user_id = $1 AND chapter_id = ANY($2)`
	rows, err := r.executor.Query(ctx, query, userID, chapterIDs)
	if err != nil {
		return nil, fmt.Errorf("repository.GetUserProgressForChapters: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var progress models.UserChapterProgress
		if err := rows.Scan(&progress.ChapterID, &progress.ProgressPercentage, &progress.VideoLastStoppedAt); err != nil {
			return nil, fmt.Errorf("repository.GetUserProgressForChapters.Scan: %w", err)
		}
		progressMap[progress.ChapterID] = progress
	}
	return progressMap, nil
}

func (r *Repository) UpdateUserProgress(ctx context.Context, userID string, chapterID int64, progress models.UpdateProgressRequest) error {
	query := `
		INSERT INTO user_chapter_progress (user_id, chapter_id, progress_percentage, video_last_stopped_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, chapter_id) DO UPDATE SET
			progress_percentage = EXCLUDED.progress_percentage,
			video_last_stopped_at = EXCLUDED.video_last_stopped_at,
			updated_at = NOW()
	`
	_, err := r.executor.Exec(ctx, query, userID, chapterID, progress.ProgressPercentage, progress.VideoLastStoppedAt)
	if err != nil {
		return fmt.Errorf("repository.UpdateUserProgress: %w", err)
	}
	return nil
}

// SaveQuizAttempt inserts a new record of a user's quiz submission into the database.
func (r *Repository) SaveQuizAttempt(ctx context.Context, attempt models.QuizAttempt) (*models.QuizAttempt, error) {
	// Marshal the user's answers map into a JSON string for the JSONB column
	answersJSON, err := json.Marshal(attempt.Answers)
	if err != nil {
		return nil, fmt.Errorf("repository.SaveQuizAttempt.Marshal: %w", err)
	}

	query := `
		INSERT INTO quiz_attempts (user_id, quiz_id, answers, score, status, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err = r.executor.QueryRow(ctx, query,
		attempt.UserID,
		attempt.QuizID,
		answersJSON,
		attempt.Score,
		attempt.Status,
		attempt.SubmittedAt,
	).Scan(&attempt.ID)

	if err != nil {
		return nil, fmt.Errorf("repository.SaveQuizAttempt.Insert: %w", err)
	}

	return &attempt, nil
}
