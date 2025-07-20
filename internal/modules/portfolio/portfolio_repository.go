package portfolio

import (
	"context"
	"errors"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBExecutor defines an interface for executing SQL queries.
type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// RepositoryInterface defines the methods for interacting with portfolio storage.
type RepositoryInterface interface {
	FindAllWorks(ctx context.Context, filters models.PortfolioFilters) ([]models.PortfolioWork, int, error)
	FindWorkByID(ctx context.Context, workID int64) (*models.PortfolioWork, error)
	GetWorkImages(ctx context.Context, workID int64) ([]models.PortfolioWorkImage, error)
	GetWorkTags(ctx context.Context, workID int64) ([]string, error)
	CreateWork(ctx context.Context, userID string, data models.CreateWorkRequest) (*models.PortfolioWork, error)
	UpdateWork(ctx context.Context, workID int64, data models.UpdateWorkRequest) (*models.PortfolioWork, error)
	DeleteWork(ctx context.Context, workID int64) error
	AddKudo(ctx context.Context, userID string, workID int64) (newKudosCount int, err error)
	RemoveKudo(ctx context.Context, userID string, workID int64) (newKudosCount int, err error)
	CheckKudos(ctx context.Context, userID string, workIDs []int64) (map[int64]bool, error)
	CheckWorkOwnership(ctx context.Context, workID int64, userID string) (bool, error)

	BeginTx(ctx context.Context) (pgx.Tx, error)
	WithTx(tx pgx.Tx) *Repository
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
	var args []interface{}
	argIdx := 1

	baseQuery := `
		FROM portfolio_works pw
		JOIN users u ON pw.user_id = u.id
	`
	if filters.Category != "" {
		baseQuery += ` JOIN portfolio_work_tags pwt ON pw.id = pwt.portfolio_work_id
		               JOIN tags t ON pwt.tag_id = t.id`
	}

	whereClause := "WHERE 1=1"
	if filters.Category != "" {
		whereClause += fmt.Sprintf(" AND t.name = $%d", argIdx)
		args = append(args, filters.Category)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(DISTINCT pw.id) " + baseQuery + whereClause
	if err := r.executor.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository.FindAllWorks.Count: %w", err)
	}
	if total == 0 {
		return []models.PortfolioWork{}, 0, nil
	}

	orderByClause := "ORDER BY pw.created_at DESC"
	if filters.Sort == "kudos" {
		orderByClause = "ORDER BY pw.kudos_count DESC, pw.created_at DESC"
	}

	selectQuery := `
		SELECT DISTINCT pw.id, pw.user_id, u.nickname as creator_nickname, pw.title,
		       pw.is_editors_choice, pw.kudos_count, pw.created_at, pw.updated_at,
		       (SELECT image_url FROM portfolio_work_images WHERE portfolio_work_id = pw.id AND is_thumbnail = TRUE LIMIT 1) as thumbnail_url
	`
	limitOffsetClause := fmt.Sprintf(" %s LIMIT $%d OFFSET $%d", orderByClause, argIdx, argIdx+1)
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	fullQuery := selectQuery + baseQuery + whereClause + limitOffsetClause
	rows, err := r.executor.Query(ctx, fullQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.FindAllWorks.Query: %w", err)
	}
	defer rows.Close()

	works := []models.PortfolioWork{}
	for rows.Next() {
		var work models.PortfolioWork
		if err := rows.Scan(
			&work.ID, &work.UserID, &work.CreatorNickname, &work.Title,
			&work.IsEditorsChoice, &work.KudosCount, &work.CreatedAt, &work.UpdatedAt,
			&work.ThumbnailURL,
		); err != nil {
			return nil, 0, fmt.Errorf("repository.FindAllWorks.Scan: %w", err)
		}
		works = append(works, work)
	}
	return works, total, nil
}

// FindWorkByID retrieves a single, detailed portfolio work.
func (r *Repository) FindWorkByID(ctx context.Context, workID int64) (*models.PortfolioWork, error) {
	var work models.PortfolioWork
	query := `
		SELECT pw.id, pw.user_id, u.nickname as creator_nickname, pw.title, pw.description,
		       pw.is_editors_choice, pw.kudos_count, pw.created_at, pw.updated_at
		FROM portfolio_works pw
		JOIN users u ON pw.user_id = u.id
		WHERE pw.id = $1
	`
	err := r.executor.QueryRow(ctx, query, workID).Scan(
		&work.ID, &work.UserID, &work.CreatorNickname, &work.Title, &work.Description,
		&work.IsEditorsChoice, &work.KudosCount, &work.CreatedAt, &work.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindWorkByID: %w", err)
	}
	return &work, nil
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

// AddKudo transactionally adds a kudo and updates the kudos count.
func (r *Repository) AddKudo(ctx context.Context, userID string, workID int64) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	insertQuery := `INSERT INTO portfolio_work_kudos (user_id, portfolio_work_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	cmdTag, err := tx.Exec(ctx, insertQuery, userID, workID)
	if err != nil {
		return 0, fmt.Errorf("repository.AddKudo.Insert: %w", err)
	}

	if cmdTag.RowsAffected() > 0 {
		updateQuery := `UPDATE portfolio_works SET kudos_count = kudos_count + 1 WHERE id = $1`
		if _, err := tx.Exec(ctx, updateQuery, workID); err != nil {
			return 0, fmt.Errorf("repository.AddKudo.Update: %w", err)
		}
	}

	var newKudosCount int
	countQuery := `SELECT kudos_count FROM portfolio_works WHERE id = $1`
	if err := tx.QueryRow(ctx, countQuery, workID).Scan(&newKudosCount); err != nil {
		return 0, fmt.Errorf("repository.AddKudo.Select: %w", err)
	}

	return newKudosCount, tx.Commit(ctx)
}
