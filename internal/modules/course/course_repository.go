package course

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"log"

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
	BeginTx(ctx context.Context) (pgx.Tx, error)
	WithTx(tx pgx.Tx) *Repository

	FindAllCourses(ctx context.Context) ([]models.Course, error)
	FindCourseByID(ctx context.Context, courseID int64) (*models.Course, error)
	FindChaptersByCourseID(ctx context.Context, courseID int64) ([]models.CourseChapter, error)
	FindChapterByID(ctx context.Context, chapterID int64) (*models.CourseChapter, error)
	FindContentBlocksByChapterID(ctx context.Context, chapterID int64) ([]models.ChapterContentBlock, error)
	FindContentBlockByID(ctx context.Context, blockID int64) (*models.ChapterContentBlock, error)
	FindAssignmentByID(ctx context.Context, assignmentID int64) (*models.Assignment, error)
	FindQuizWithAnswersByID(ctx context.Context, quizID int64) (*models.Quiz, error)

	CheckUserEnrollment(ctx context.Context, userID string, courseID int64) (bool, error)
	EnrollUserInCourse(ctx context.Context, userID string, courseID int64) error
	FindEnrolledCoursesByUserID(ctx context.Context, userID string) ([]models.EnrolledCourseResponse, error)

	UpdateLastVisitedAt(ctx context.Context, userID string, courseID int64) error
	SavePassiveBlockCompletion(ctx context.Context, userID string, blockID int64) error
	SaveVideoProgress(ctx context.Context, userID string, blockID, lastStoppedAt int64) error
	SubmitAssignment(ctx context.Context, submission models.AssignmentSubmission) (*models.AssignmentSubmission, error)
	SaveQuizAttempt(ctx context.Context, attempt models.QuizAttempt) (*models.QuizAttempt, error)

	GetUserProgressForChapters(ctx context.Context, userID string, chapterIDs []int64) (map[int64]models.UserChapterProgress, error)
	CalculateChapterProgress(ctx context.Context, userID string, chapterID int64) error
}

type Repository struct {
	db       *pgxpool.Pool
	executor DBExecutor
}

type Scannable interface {
	Scan(dest ...any) error
}

func NewRepository(db *pgxpool.Pool) RepositoryInterface {
	return &Repository{db: db, executor: db}
}

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{db: r.db, executor: tx}
}

func (r *Repository) scanCourse(row Scannable) (*models.Course, error) {
	var course models.Course
	var instructorID sql.NullString

	err := row.Scan(
		&course.ID,
		&course.Title,
		&course.Description,
		&instructorID,
		&course.ThumbnailURL,
		&course.CreatedAt,
		&course.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if instructorID.Valid {
		course.InstructorID = &instructorID.String
	} else {
		course.InstructorID = nil
	}

	return &course, nil
}

func (r *Repository) FindAllCourses(ctx context.Context) ([]models.Course, error) {
	courses := []models.Course{}
	query := `SELECT id, title, description, instructor_id, thumbnail_url, created_at, updated_at FROM courses ORDER BY created_at ASC`
	rows, err := r.executor.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository.FindAllCourses.Query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		course, err := r.scanCourse(rows)
		if err != nil {
			return nil, fmt.Errorf("repository.FindAllCourses.Scan: %w", err)
		}
		courses = append(courses, *course)
	}
	return courses, nil
}

func (r *Repository) FindCourseByID(ctx context.Context, courseID int64) (*models.Course, error) {
	query := `SELECT id, title, description, thumbnail_url, created_at, updated_at FROM courses WHERE id = $1`
	row := r.executor.QueryRow(ctx, query, courseID)
	course, err := r.scanCourse(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindCourseByID: %w", err)
	}
	return course, nil
}

func (r *Repository) FindChaptersByCourseID(ctx context.Context, courseID int64) ([]models.CourseChapter, error) {
	chapters := []models.CourseChapter{}

	query := `
		SELECT 
			cc.id, cc.title, cc.display_order, cc.available_for_guests,
			COUNT(CASE WHEN ccb.type = 'video' THEN 1 END) as video_count,
			COUNT(CASE WHEN ccb.type = 'reading' THEN 1 END) as reading_count,
			COUNT(CASE WHEN ccb.type = 'quiz' THEN 1 END) as quiz_count,
			COUNT(CASE WHEN ccb.type = 'assignment' THEN 1 END) as assignment_count
		FROM course_chapters cc
		LEFT JOIN chapter_content_blocks ccb ON cc.id = ccb.chapter_id
		WHERE cc.course_id = $1
		GROUP BY cc.id
		ORDER BY cc.display_order ASC
	`
	rows, err := r.executor.Query(ctx, query, courseID)
	if err != nil {
		return nil, fmt.Errorf("repository.FindChaptersByCourseID.Query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var chapter models.CourseChapter
		if err := rows.Scan(
			&chapter.ID, &chapter.Title, &chapter.DisplayOrder, &chapter.AvailableForGuests,
			&chapter.ContentBlockCount.VideoCount, &chapter.ContentBlockCount.ReadingCount,
			&chapter.ContentBlockCount.QuizCount, &chapter.ContentBlockCount.AssignmentCount,
		); err != nil {
			return nil, fmt.Errorf("repository.FindChaptersByCourseID.Scan: %w", err)
		}
		chapters = append(chapters, chapter)
	}

	return chapters, nil
}

func (r *Repository) FindChapterByID(ctx context.Context, chapterID int64) (*models.CourseChapter, error) {
	var chapter models.CourseChapter
	query := `SELECT id, course_id, title, available_for_guests FROM course_chapters WHERE id = $1`
	err := r.executor.QueryRow(ctx, query, chapterID).Scan(
		&chapter.ID, &chapter.CourseID, &chapter.Title, &chapter.AvailableForGuests,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindChapterByID: %w", err)
	}
	return &chapter, nil
}

func (r *Repository) FindContentBlocksByChapterID(ctx context.Context, chapterID int64) ([]models.ChapterContentBlock, error) {
	blocks := []models.ChapterContentBlock{}
	query := `
		SELECT id, chapter_id, type, content, display_order
		FROM chapter_content_blocks
		WHERE chapter_id = $1
		ORDER BY display_order ASC
	`
	rows, err := r.executor.Query(ctx, query, chapterID)
	if err != nil {
		return nil, fmt.Errorf("repository.FindContentBlocksByChapterID: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var block models.ChapterContentBlock
		var contentJSON []byte // Scan the raw JSONB into a byte slice

		if err := rows.Scan(&block.ID, &block.ChapterID, &block.Type, &contentJSON, &block.DisplayOrder); err != nil {
			return nil, fmt.Errorf("repository.FindContentBlocksByChapterID.Scan: %w", err)
		}

		// Unmarshal into the correct struct based on the 'type' field.
		switch block.Type {
		case "video":
			var videoContent models.VideoContent
			if err := json.Unmarshal(contentJSON, &videoContent); err != nil {
				log.Printf("WARN: could not unmarshal video content for block %d: %v", block.ID, err)
				continue // Skip this block if content is malformed
			}
			block.Content = videoContent
		case "reading":
			var readingContent models.ReadingContent
			if err := json.Unmarshal(contentJSON, &readingContent); err != nil {
				log.Printf("WARN: could not unmarshal reading content for block %d: %v", block.ID, err)
				continue
			}
			block.Content = readingContent
		case "assignment":
			var assignmentContent models.AssignmentContent
			if err := json.Unmarshal(contentJSON, &assignmentContent); err != nil {
				log.Printf("WARN: could not unmarshal assignment content for block %d: %v", block.ID, err)
				continue
			}
			block.Content = assignmentContent
		case "quiz":
			var quizContent models.QuizContent
			if err := json.Unmarshal(contentJSON, &quizContent); err != nil {
				log.Printf("WARN: could not unmarshal quiz content for block %d: %v", block.ID, err)
				continue
			}
			block.Content = quizContent
		default:
			log.Printf("WARN: unknown content block type '%s' for block %d", block.Type, block.ID)
			continue
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}

func (r *Repository) FindContentBlockByID(ctx context.Context, blockID int64) (*models.ChapterContentBlock, error) {
	var block models.ChapterContentBlock

	query := `
		SELECT ccb.id, ccb.chapter_id, ccb.type, ccb.content, ccb.display_order, cc.course_id
		FROM chapter_content_blocks ccb
		JOIN course_chapters cc ON ccb.chapter_id = cc.id
		WHERE ccb.id = $1
	`
	var contentJSON []byte
	err := r.executor.QueryRow(ctx, query, blockID).Scan(
		&block.ID, &block.ChapterID, &block.Type, &contentJSON, &block.DisplayOrder, &block.CourseID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindContentBlockByID: %w", err)
	}

	// Unmarshal logic
	switch block.Type {
	case "video":
		var videoContent models.VideoContent
		if err := json.Unmarshal(contentJSON, &videoContent); err != nil {
			log.Printf("WARN: could not unmarshal video content for block %d: %v", block.ID, err)
		} else {
			block.Content = videoContent
		}
	case "reading":
		var readingContent models.ReadingContent
		if err := json.Unmarshal(contentJSON, &readingContent); err != nil {
			log.Printf("WARN: could not unmarshal reading content for block %d: %v", block.ID, err)
		} else {
			block.Content = readingContent
		}
	case "assignment":
		var asgnContent models.AssignmentContent
		if err := json.Unmarshal(contentJSON, &asgnContent); err != nil {
			log.Printf("WARN: could not unmarshal assignment content for block %d: %v", block.ID, err)
		} else {
			block.Content = asgnContent
		}
	case "quiz":
		var quizContent models.QuizContent
		if err := json.Unmarshal(contentJSON, &quizContent); err != nil {
			log.Printf("WARN: could not unmarshal quiz content for block %d: %v", block.ID, err)
		} else {
			block.Content = quizContent
		}
	}

	return &block, nil
}

func (r *Repository) FindAssignmentByID(ctx context.Context, assignmentID int64) (*models.Assignment, error) {
	var asgn models.Assignment
	var deadline sql.NullTime

	query := `SELECT id, title, description, attachment_urls, apply_deadline, deadline
	FROM assignments WHERE id = $1`
	err := r.executor.QueryRow(ctx, query, assignmentID).Scan(
		&asgn.ID, &asgn.Title, &asgn.AttachmentURLs, &asgn.ApplyDeadline, &deadline,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindAssignmentByID: %w", err)
	}
	if deadline.Valid {
		asgn.Deadline = &deadline.Time
	} else {
		asgn.Deadline = nil
	}

	return &asgn, nil
}

// FindQuizWithAnswersByID retrieves the full quiz structure, including correct answers, from the database.
// This should only be called by the service layer for scoring, not sent directly to the client.
func (r *Repository) FindQuizWithAnswersByID(ctx context.Context, quizID int64) (*models.Quiz, error) {
	var quiz models.Quiz
	var questionsJSON []byte // scan the JSONB data into a byte slice first

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

func (r *Repository) FindEnrolledCoursesByUserID(ctx context.Context, userID string) ([]models.EnrolledCourseResponse, error) {
	courses := []models.EnrolledCourseResponse{}
	query := `
		SELECT
			c.id,
			c.title,
			c.thumbnail_url,
			ue.last_visited_at
		FROM courses c
		JOIN user_enrollments ue ON c.id = ue.course_id
		WHERE ue.user_id = $1
		ORDER BY ue.last_visited_at DESC
	`
	rows, err := r.executor.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("repository.FindEnrolledCoursesByUserID.Query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var course models.EnrolledCourseResponse
		if err := rows.Scan(
			&course.ID,
			&course.Title,
			&course.ThumbnailURL,
			&course.LastVisitedAt,
		); err != nil {
			return nil, fmt.Errorf("repository.FindEnrolledCoursesByUserID.Scan: %w", err)
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.FindEnrolledCoursesByUserID.RowsErr: %w", err)
	}

	return courses, nil
}

func (r *Repository) UpdateLastVisitedAt(ctx context.Context, userID string, courseID int64) error {
	query := `
		UPDATE user_enrollments
		SET last_visited_at = NOW()
		WHERE user_id = $1 AND course_id = $2
	`
	// Use Exec for statements that don't return rows.
	cmdTag, err := r.executor.Exec(ctx, query, userID, courseID)
	if err != nil {
		// In a background task, we should log the error rather than returning it,
		// as there's no calling function to handle it.
		// However, for a clean repository method, we return the error.
		// The service layer, which calls this in a goroutine, will just let it finish.
		return fmt.Errorf("repository.UpdateLastVisitedAt: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		// This means the enrollment record wasn't found. This is an unexpected state
		// if this function is called after an enrollment check, so it's worth noting.
		return fmt.Errorf("repository.UpdateLastVisitedAt: no enrollment found for user %s in course %d", userID, courseID)
	}

	return nil
}

func (r *Repository) SavePassiveBlockCompletion(ctx context.Context, userID string, blockID int64) error {
	query := `
		INSERT INTO user_content_block_progress (user_id, content_block_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, content_block_id) DO NOTHING
	`
	_, err := r.executor.Exec(ctx, query, userID, blockID)
	if err != nil {
		return fmt.Errorf("repository.SavePassiveBlockCompletion: %w", err)
	}
	return nil
}

func (r *Repository) SaveVideoProgress(ctx context.Context, userID string, blockID, lastStoppedAt int64) error {
	query := `
		INSERT INTO user_video_progress (user_id, content_block_id, last_stopped_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, content_block_id) DO UPDATE SET
			last_stopped_at = EXCLUDED.last_stopped_at,
			updated_at = NOW()
	`
	_, err := r.executor.Exec(ctx, query, userID, blockID, lastStoppedAt)
	if err != nil {
		return fmt.Errorf("repository.SaveVideoProgress: %w", err)
	}
	return nil
}

func (r *Repository) SubmitAssignment(ctx context.Context, submission models.AssignmentSubmission) (*models.AssignmentSubmission, error) {
	answersJSON, err := json.Marshal(submission.Answers)
	if err != nil {
		return nil, fmt.Errorf("repository.SubmitAssignment.Marshal: %w", err)
	}

	query := `
		INSERT INTO assignment_submissions (user_id, assignment_id, answers, status, submitted_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err = r.executor.QueryRow(ctx, query,
		submission.UserID,
		submission.AssignmentID,
		answersJSON,
		submission.Status,
		submission.SubmittedAt,
	).Scan(&submission.ID)

	if err != nil {
		return nil, fmt.Errorf("repository.SubmitAssignment.Insert: %w", err)
	}

	return &submission, nil
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

func (r *Repository) GetUserProgressForChapters(ctx context.Context, userID string, chapterIDs []int64) (map[int64]models.UserChapterProgress, error) {
	progressMap := make(map[int64]models.UserChapterProgress)
	if userID == "" || len(chapterIDs) == 0 {
		return progressMap, nil
	}

	query := `SELECT chapter_id, progress_percentage, completed_at, updated_at FROM user_chapter_progress WHERE user_id = $1 AND chapter_id = ANY($2)`
	rows, err := r.executor.Query(ctx, query, userID, chapterIDs)
	if err != nil {
		return nil, fmt.Errorf("repository.GetUserProgressForChapters: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var progress models.UserChapterProgress
		if err := rows.Scan(
			&progress.ChapterID,
			&progress.ProgressPercentage,
			&progress.CompletedAt,
			&progress.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository.GetUserProgressForChapters.Scan: %w", err)
		}
		progressMap[progress.ChapterID] = progress
	}
	return progressMap, nil
}

func (r *Repository) CalculateChapterProgress(ctx context.Context, userID string, chapterID int64) error {
	// This query is complex. It calculates the number of completable blocks in a chapter
	// and the number of blocks the user has completed, then updates the user_chapter_progress table.
	query := `
		WITH ChapterBlockCounts AS (
			-- 1. Count total completable blocks in the chapter
			SELECT COUNT(*) as total_blocks
			FROM chapter_content_blocks
			WHERE chapter_id = $2
		),
		UserCompletedCounts AS (
			-- 2. Count completed blocks for the user in this chapter
			SELECT COUNT(*) as completed_blocks
			FROM (
				-- Completed passive blocks (videos, readings)
				SELECT ccb.id FROM user_content_block_progress ucbp
				JOIN chapter_content_blocks ccb ON ucbp.content_block_id = ccb.id
				WHERE ucbp.user_id = $1 AND ccb.chapter_id = $2
				UNION
				-- Completed quizzes
				SELECT ccb.id FROM quiz_attempts qa
				JOIN quizzes q ON qa.quiz_id = q.id
				JOIN chapter_content_blocks ccb ON (ccb.content->>'quiz_id')::bigint = q.id
				WHERE qa.user_id = $1 AND ccb.chapter_id = $2
				UNION
				-- Completed assignments
				SELECT ccb.id FROM assignment_submissions asub
				JOIN assignments a ON asub.assignment_id = a.id
				JOIN chapter_content_blocks ccb ON (ccb.content->>'assignment_id')::bigint = a.id
				WHERE asub.user_id = $1 AND ccb.chapter_id = $2
			) as completed
		),
		NewProgress AS (
			-- 3. Calculate the new percentage
			SELECT
				CASE
					WHEN cbc.total_blocks > 0 THEN (ucc.completed_blocks::decimal / cbc.total_blocks * 100)::integer
					ELSE 0
				END as new_percentage,
				CASE
					WHEN ucc.completed_blocks = cbc.total_blocks AND cbc.total_blocks > 0 THEN NOW()
					ELSE NULL
				END as completed_at_time
			FROM ChapterBlockCounts cbc, UserCompletedCounts ucc
		)
		-- 4. Update the user_chapter_progress table with the new percentage
		INSERT INTO user_chapter_progress (user_id, chapter_id, progress_percentage, completed_at)
		SELECT $1, $2, np.new_percentage, np.completed_at_time
		FROM NewProgress np
		ON CONFLICT (user_id, chapter_id) DO UPDATE SET
			progress_percentage = EXCLUDED.progress_percentage,
			-- Only set completed_at if it's not already set, to preserve the original completion time
			completed_at = COALESCE(user_chapter_progress.completed_at, EXCLUDED.completed_at),
			updated_at = NOW();
	`
	_, err := r.executor.Exec(ctx, query, userID, chapterID)
	if err != nil {
		return fmt.Errorf("repository.CalculateChapterProgress: %w", err)
	}
	return nil
}
