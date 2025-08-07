package portfolio

import (
	"context"
	"errors"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

// DBExecutor defines an interface for executing SQL queries.
type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

// RepositoryInterface defines the methods for interacting with portfolio storage.
type RepositoryInterface interface {
	BeginTx(ctx context.Context) (pgx.Tx, error)
	WithTx(tx pgx.Tx) *Repository

	FindAllWorks(ctx context.Context, filters models.PortfolioFilters) ([]models.PortfolioWork, int, error)
	FindWorkByID(ctx context.Context, workID int64) (*models.PortfolioWork, error)
	GetWorkCountByUserID(ctx context.Context, userID string) (int, error)
	GetWorkImages(ctx context.Context, workID int64) ([]models.PortfolioWorkImage, error)
	GetWorkTags(ctx context.Context, workID int64) ([]string, error)
	CreateWork(ctx context.Context, userID string, data models.CreateWorkRequest) (*models.PortfolioWork, error)
	UpdateWork(ctx context.Context, workID int64, data models.UpdateWorkRequest) (*models.PortfolioWork, error)
	DeleteWork(ctx context.Context, workID int64) error
	Upvote(ctx context.Context, userID string, workID int64) (newUpvotesCount int, err error)
	Downvote(ctx context.Context, userID string, workID int64) (newUpvotesCount int, err error)
	IsWorkUpvotedByUser(ctx context.Context, userID string, workID int64) (bool, error)
	CheckUpvotes(ctx context.Context, userID string, workIDs []int64) (map[int64]bool, error)
	CheckWorkOwnership(ctx context.Context, workID int64, userID string) (bool, error)
}

// Repository provides access to the portfolio storage.
type Repository struct {
	db       *pgxpool.Pool
	executor DBExecutor
}

// NewRepository creates a new portfolio repository.
func NewRepository(db *pgxpool.Pool) RepositoryInterface {
	return &Repository{db: db, executor: db}
}

// BeginTx starts a new database transaction.
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// WithTx returns a new repository instance scoped to the provided transaction.
func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{db: r.db, executor: tx}
}

// FindAllWorks retrieves a paginated and filtered list of portfolio works.
func (r *Repository) FindAllWorks(ctx context.Context, filters models.PortfolioFilters) ([]models.PortfolioWork, int, error) {
	var args []any
	var whereClauses []string
	var joinClause strings.Builder

	// Start with a base WHERE clause that is always true
	whereClauses = append(whereClauses, "1=1")

	if len(filters.Tags) > 0 {
		// Add the JOIN clause needed for tag filtering
		joinClause.WriteString(` JOIN portfolio_work_tags pwt ON pw.id = pwt.portfolio_work_id
		                         JOIN tags t ON pwt.tag_id = t.id`)
		// Add the WHERE clause for tags using ANY for efficiency
		args = append(args, filters.Tags)
		whereClauses = append(whereClauses, fmt.Sprintf("t.name = ANY($%d)", len(args)))
	}

	finalWhereClause := "WHERE " + strings.Join(whereClauses, " AND ")

	baseQuery := fmt.Sprintf(`FROM portfolio_works pw JOIN users u ON pw.user_id = u.id %s`, joinClause.String())

	var total int
	countQuery := "SELECT COUNT(DISTINCT pw.id) " + baseQuery + finalWhereClause
	if err := r.executor.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository.FindAllWorks.Count: %w", err)
	}
	if total == 0 {
		return []models.PortfolioWork{}, 0, nil
	}

	orderByClause := "ORDER BY pw.created_at DESC"
	if filters.Sort == "upvotes" {
		orderByClause = "ORDER BY pw.upvotes_count DESC, pw.created_at DESC"
	}

	// Add pagination arguments
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)
	limitOffsetClause := fmt.Sprintf(" %s LIMIT $%d OFFSET $%d", orderByClause, len(args)-1, len(args))

	selectQuery := `
		SELECT DISTINCT pw.id, pw.user_id, u.nickname as creator_nickname, pw.title,
		       pw.is_editors_choice, pw.upvotes_count, pw.created_at, pw.updated_at,
		       (SELECT image_url FROM portfolio_work_images WHERE portfolio_work_id = pw.id AND is_thumbnail = TRUE LIMIT 1) as thumbnail_url
	`

	fullQuery := selectQuery + baseQuery + finalWhereClause + limitOffsetClause
	rows, err := r.executor.Query(ctx, fullQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.FindAllWorks.Query: %w", err)
	}

	works, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.PortfolioWork])
	if err != nil {
		return nil, 0, fmt.Errorf("repository.FindAllWorks.Scan: %w", err)
	}

	return works, total, nil
}

// FindWorkByID retrieves a single, detailed portfolio work.
func (r *Repository) FindWorkByID(ctx context.Context, workID int64) (*models.PortfolioWork, error) {
	var work models.PortfolioWork
	query := `
		SELECT pw.id, pw.user_id, u.nickname as creator_nickname, pw.title, pw.description,
		       pw.is_editors_choice, pw.upvotes_count, pw.created_at, pw.updated_at
		FROM portfolio_works pw
		JOIN users u ON pw.user_id = u.id
		WHERE pw.id = $1
	`
	err := r.executor.QueryRow(ctx, query, workID).Scan(
		&work.ID, &work.UserID, &work.CreatorNickname, &work.Title, &work.Description,
		&work.IsEditorsChoice, &work.UpvotesCount, &work.CreatedAt, &work.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindWorkByID: %w", err)
	}
	return &work, nil
}

// GetWorkCountByUserID counts the number of works a user has created.
func (r *Repository) GetWorkCountByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM portfolio_works WHERE user_id = $1`
	err := r.executor.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// GetWorkImages retrieves all images for a given portfolio work.
func (r *Repository) GetWorkImages(ctx context.Context, workID int64) ([]models.PortfolioWorkImage, error) {
	query := `SELECT id, image_url, is_thumbnail, caption, display_order FROM portfolio_work_images WHERE portfolio_work_id = $1 ORDER BY id`
	rows, err := r.executor.Query(ctx, query, workID)
	if err != nil {
		return nil, fmt.Errorf("repository.GetWorkImages.Query: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.PortfolioWorkImage])
}

// GetWorkTags retrieves all tag names for a given portfolio work.
func (r *Repository) GetWorkTags(ctx context.Context, workID int64) ([]string, error) {
	query := `
		SELECT t.name
		FROM tags t
		JOIN portfolio_work_tags pwt ON t.id = pwt.tag_id
		WHERE pwt.portfolio_work_id = $1
		ORDER BY t.name
	`
	rows, err := r.executor.Query(ctx, query, workID)
	if err != nil {
		return nil, fmt.Errorf("repository.GetWorkTags.Query: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// linkTagsToWork is a helper function to be used within a transaction.
func (r *Repository) linkTagsToWork(ctx context.Context, workID int64, tags []string) error {
	// First, find or create all tags and get their IDs
	tagIDs := make([]int64, 0, len(tags))
	for _, tagName := range tags {
		var tagID int64
		// Use ON CONFLICT to atomically find or create a tag
		query := `
			WITH ins AS (
				INSERT INTO tags (name) VALUES ($1)
				ON CONFLICT (name) DO NOTHING
				RETURNING id
			)
			SELECT id FROM ins
			UNION ALL
			SELECT id FROM tags WHERE name = $1 LIMIT 1
		`
		err := r.executor.QueryRow(ctx, query, tagName).Scan(&tagID)
		if err != nil {
			return fmt.Errorf("linking tag '%s': %w", tagName, err)
		}
		tagIDs = append(tagIDs, tagID)
	}

	// Now, bulk insert the associations into the junction table
	tagLinkRows := make([][]interface{}, len(tagIDs))
	for i, tagID := range tagIDs {
		tagLinkRows[i] = []interface{}{workID, tagID}
	}

	_, err := r.executor.CopyFrom(
		ctx,
		pgx.Identifier{"portfolio_work_tags"},
		[]string{"portfolio_work_id", "tag_id"},
		pgx.CopyFromRows(tagLinkRows),
	)
	return err
}

// CreateWork transactionally creates a new portfolio work, its images, and its tags.
func (r *Repository) CreateWork(ctx context.Context, userID string, data models.CreateWorkRequest) (*models.PortfolioWork, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	repoTx := r.WithTx(tx)

	// 1. Insert the main portfolio work record
	var workID int64
	workQuery := `
		INSERT INTO portfolio_works (user_id, title, description)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	if err := repoTx.executor.QueryRow(ctx, workQuery, userID, data.Title, data.Description).Scan(&workID); err != nil {
		return nil, fmt.Errorf("repository.CreateWork.InsertWork: %w", err)
	}

	// 2. Bulk insert images using CopyFrom for high performance
	if len(data.ImageURLs) > 0 {
		imageRows := make([][]any, len(data.ImageURLs))
		for i, url := range data.ImageURLs {
			isThumbnail := data.ThumbnailURLIndex != nil && *data.ThumbnailURLIndex == i
			imageRows[i] = []any{workID, url, isThumbnail, "", i} // caption is empty on create
		}
		_, err := repoTx.executor.CopyFrom(
			ctx,
			pgx.Identifier{"portfolio_work_images"},
			[]string{"portfolio_work_id", "image_url", "is_thumbnail", "caption", "display_order"},
			pgx.CopyFromRows(imageRows),
		)
		if err != nil {
			return nil, fmt.Errorf("repository.CreateWork.CopyFromImages: %w", err)
		}
	}

	// 3. Handle tags: find existing or create new ones, then link them
	if len(data.Tags) > 0 {
		if err := repoTx.linkTagsToWork(ctx, workID, data.Tags); err != nil {
			return nil, fmt.Errorf("repository.CreateWork.LinkTags: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// 4. Fetch the newly created work to return it
	return repoTx.FindWorkByID(ctx, workID)
}

func (r *Repository) UpdateWork(ctx context.Context, workID int64, data models.UpdateWorkRequest) (*models.PortfolioWork, error) {
	// This would also be a transactional operation.
	// 1. Begin Tx
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	repoTx := r.WithTx(tx)

	// 2. Update the portfolio_works table with new title/description
	updateQuery := `
		UPDATE portfolio_works
		SET title = COALESCE($1, title), description = COALESCE($2, description), updated_at = $3
		WHERE id = $4
	`
	if _, err := repoTx.executor.Exec(ctx, updateQuery, data.Title, data.Description, time.Now(), workID); err != nil {
		return nil, fmt.Errorf("repository.UpdateWork.UpdateDetails: %w", err)
	}

	// 3. Replace images: Delete old ones, then insert the new set.
	if data.Images != nil {
		if _, err := repoTx.executor.Exec(ctx, "DELETE FROM portfolio_work_images WHERE portfolio_work_id = $1", workID); err != nil {
			return nil, fmt.Errorf("repository.UpdateWork.DeleteImages: %w", err)
		}
		if len(data.Images) > 0 {
			imageRows := make([][]any, len(data.Images))
			for i, img := range data.Images {
				imageRows[i] = []any{workID, img.ImageURL, img.IsThumbnail, img.Caption, i}
			}
			_, err := repoTx.executor.CopyFrom(
				ctx,
				pgx.Identifier{"portfolio_work_images"},
				[]string{"portfolio_work_id", "image_url", "is_thumbnail", "caption", "display_order"},
				pgx.CopyFromRows(imageRows),
			)
			if err != nil {
				return nil, fmt.Errorf("repository.UpdateWork.CopyFromImages: %w", err)
			}
		}
	}

	// 4. Replace tags: Delete old associations, then link the new set.
	if data.Tags != nil {
		if _, err := repoTx.executor.Exec(ctx, "DELETE FROM portfolio_work_tags WHERE portfolio_work_id = $1", workID); err != nil {
			return nil, fmt.Errorf("repository.UpdateWork.DeleteTags: %w", err)
		}
		if len(data.Tags) > 0 {
			if err := repoTx.linkTagsToWork(ctx, workID, data.Tags); err != nil {
				return nil, fmt.Errorf("repository.UpdateWork.LinkTags: %w", err)
			}
		}
	}

	// 5. Commit Tx
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// 6. Return the updated work
	return repoTx.FindWorkByID(ctx, workID)
}

// DeleteWork deletes a portfolio work. The ON DELETE CASCADE in the DB handles related data.
func (r *Repository) DeleteWork(ctx context.Context, workID int64) error {
	query := `DELETE FROM portfolio_works WHERE id = $1`
	cmdTag, err := r.executor.Exec(ctx, query, workID)
	if err != nil {
		return fmt.Errorf("repository.DeleteWork: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

// CheckWorkOwnership verifies if a user is the creator of a portfolio work.
func (r *Repository) CheckWorkOwnership(ctx context.Context, workID int64, userID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM portfolio_works WHERE id = $1 AND user_id = $2)`
	err := r.executor.QueryRow(ctx, query, workID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("repository.CheckWorkOwnership: %w", err)
	}
	return exists, nil
}

// CheckUpvotes checks which of a list of work IDs a user has upvoted.
func (r *Repository) CheckUpvotes(ctx context.Context, userID string, workIDs []int64) (map[int64]bool, error) {
	if len(workIDs) == 0 || userID == "" {
		return map[int64]bool{}, nil
	}

	query := `SELECT portfolio_work_id FROM portfolio_work_upvotes WHERE user_id = $1 AND portfolio_work_id = ANY($2)`
	rows, err := r.executor.Query(ctx, query, userID, workIDs)
	if err != nil {
		return nil, fmt.Errorf("repository.CheckUpvotes.Query: %w", err)
	}

	upvoteMap := make(map[int64]bool, len(workIDs))
	for rows.Next() {
		var workID int64
		if err := rows.Scan(&workID); err != nil {
			return nil, fmt.Errorf("repository.CheckUpvotes.Scan: %w", err)
		}
		upvoteMap[workID] = true
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("repository.CheckUpvotes.RowsErr: %w", rows.Err())
	}

	return upvoteMap, nil
}

func (r *Repository) IsWorkUpvotedByUser(ctx context.Context, userID string, workID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM portfolio_work_upvotes WHERE user_id = $1 AND work_id = $2)`
	err := r.executor.QueryRow(ctx, query, userID, workID).Scan(&exists)
	return exists, err
}

// Upvote transactionally upvotes and updates the upvotes count.
func (r *Repository) Upvote(ctx context.Context, userID string, workID int64) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	insertQuery := `INSERT INTO portfolio_work_upvotes (user_id, portfolio_work_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	cmdTag, err := tx.Exec(ctx, insertQuery, userID, workID)
	if err != nil {
		return 0, fmt.Errorf("repository.Upvote.Insert: %w", err)
	}

	if cmdTag.RowsAffected() > 0 {
		updateQuery := `UPDATE portfolio_works SET upvotes_count = upvotes_count + 1 WHERE id = $1`
		if _, err := tx.Exec(ctx, updateQuery, workID); err != nil {
			return 0, fmt.Errorf("repository.Upvote.Update: %w", err)
		}
	}

	var newUpvotesCount int
	countQuery := `SELECT upvotes_count FROM portfolio_works WHERE id = $1`
	if err := tx.QueryRow(ctx, countQuery, workID).Scan(&newUpvotesCount); err != nil {
		return 0, fmt.Errorf("repository.Upvote.Select: %w", err)
	}

	return newUpvotesCount, tx.Commit(ctx)
}

// Downvote transactionally removes upvote and updates the upvotes count.
func (r *Repository) Downvote(ctx context.Context, userID string, workID int64) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	deleteQuery := `DELETE FROM portfolio_work_upvotes WHERE user_id = $1 AND portfolio_work_id = $2`
	cmdTag, err := tx.Exec(ctx, deleteQuery, userID, workID)
	if err != nil {
		return 0, fmt.Errorf("repository.Upvote.Delete: %w", err)
	}

	if cmdTag.RowsAffected() > 0 {
		updateQuery := `UPDATE portfolio_works SET upvotes_count = upvotes_count - 1 WHERE id = $1`
		if _, err := tx.Exec(ctx, updateQuery, workID); err != nil {
			return 0, fmt.Errorf("repository.Downvote.Update: %w", err)
		}
	}

	var newUpvotesCount int
	countQuery := `SELECT upvotes_count FROM portfolio_works WHERE id = $1`
	if err := tx.QueryRow(ctx, countQuery, workID).Scan(&newUpvotesCount); err != nil {
		return 0, fmt.Errorf("repository.Downvote.Select: %w", err)
	}

	return newUpvotesCount, tx.Commit(ctx)
}
