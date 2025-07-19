package gallery

import (
	"context"
	"errors"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBExecutor defines an interface for executing SQL queries, implemented by both *pgxpool.Pool and pgx.Tx.
type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type RepositoryInterface interface {
	FindAllArtworks(ctx context.Context, filters models.ArtworkFilters) ([]models.Artwork, int, error)
	FindArtworkByID(ctx context.Context, artworkID int64) (*models.Artwork, error)
	GetArtworkImages(ctx context.Context, artworkID int64) ([]models.ArtworkImage, error)
	GetArtworkTags(ctx context.Context, artworkID int64) ([]string, error)
	FindAllArtists(ctx context.Context, page, limit int) ([]models.Artist, int, error)
	FindArtistByID(ctx context.Context, artistID int64) (*models.Artist, error)
	FindAllCategories(ctx context.Context) ([]string, error)
	CheckFavorites(ctx context.Context, userID string, artworkIDs []int64) (map[int64]bool, error)
	AddFavorite(ctx context.Context, userID string, artworkID int64) error
	RemoveFavorite(ctx context.Context, userID string, artworkID int64) error

	BeginTx(ctx context.Context) (pgx.Tx, error)
	WithTx(tx pgx.Tx) *Repository
}

type Repository struct {
	db       *pgxpool.Pool
	executor DBExecutor
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

func (r *Repository) FindAllArtworks(ctx context.Context, filters models.ArtworkFilters) ([]models.Artwork, int, error) {
	// Use squirrel or another query builder for more complex dynamic queries.
	// This is a manual string building example.
	baseQuery := `
		FROM artworks a
		LEFT JOIN artists ar ON a.artist_id = ar.id
	`
	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if filters.Category != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("a.category = $%d", argIdx))
		args = append(args, filters.Category)
		argIdx++
	}
	if filters.ArtistID > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("a.artist_id = $%d", argIdx))
		args = append(args, filters.ArtistID)
		argIdx++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Get total count with filters
	var total int
	countQuery := "SELECT COUNT(a.id) " + baseQuery + whereClause
	err := r.executor.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.FindAllArtworks.Count: %w", err)
	}

	if total == 0 {
		return []models.Artwork{}, 0, nil
	}

	// Get paginated data
	selectQuery := `
		SELECT a.id, a.title, a.category, a.thumbnail_url, a.created_at, a.artist_id, ar.name as artist_name
	`
	limitOffsetClause := fmt.Sprintf(" ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	fullQuery := selectQuery + baseQuery + whereClause + limitOffsetClause
	rows, err := r.executor.Query(ctx, fullQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.FindAllArtworks.Query: %w", err)
	}
	defer rows.Close()

	artworks := []models.Artwork{}
	for rows.Next() {
		var art models.Artwork
		if err := rows.Scan(&art.ID, &art.Title, &art.Category, &art.ThumbnailURL, &art.CreatedAt, &art.ArtistID, &art.ArtistName); err != nil {
			return nil, 0, fmt.Errorf("repository.FindAllArtworks.Scan: %w", err)
		}
		artworks = append(artworks, art)
	}

	return artworks, total, nil
}

func (r *Repository) FindArtworkByID(ctx context.Context, artworkID int64) (*models.Artwork, error) {
	var art models.Artwork
	query := `
		SELECT a.id, a.title, a.category, a.thumbnail_url, a.created_at, a.artist_id, ar.name as artist_name,
		       a.description, a.creation_year, a.dimensions, a.introduction
		FROM artworks a
		LEFT JOIN artists ar ON a.artist_id = ar.id
		WHERE a.id = $1
	`
	err := r.executor.QueryRow(ctx, query, artworkID).Scan(
		&art.ID, &art.Title, &art.Category, &art.ThumbnailURL, &art.CreatedAt, &art.ArtistID, &art.ArtistName,
		&art.Description, &art.CreationYear, &art.Dimensions, &art.Introduction,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindArtworkByID: %w", err)
	}
	return &art, nil
}

func (r *Repository) GetArtworkImages(ctx context.Context, artworkID int64) ([]models.ArtworkImage, error) {
	images := []models.ArtworkImage{}
	query := `SELECT id, image_url, caption FROM artwork_images WHERE artwork_id = $1 ORDER BY display_order ASC`
	rows, err := r.executor.Query(ctx, query, artworkID)
	if err != nil {
		return nil, fmt.Errorf("repository.GetArtworkImages: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var img models.ArtworkImage
		if err := rows.Scan(&img.ID, &img.ImageURL, &img.Caption); err != nil {
			return nil, fmt.Errorf("repository.GetArtworkImages.Scan: %w", err)
		}
		images = append(images, img)
	}
	return images, nil
}

func (r *Repository) GetArtworkTags(ctx context.Context, artworkID int64) ([]string, error) {
	tags := []string{}
	query := `
		SELECT t.name FROM tags t
		JOIN artwork_tags at ON t.id = at.tag_id
		WHERE at.artwork_id = $1
	`
	rows, err := r.executor.Query(ctx, query, artworkID)
	if err != nil {
		return nil, fmt.Errorf("repository.GetArtworkTags: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("repository.GetArtworkTags.Scan: %w", err)
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (r *Repository) FindAllArtists(ctx context.Context, page, limit int) ([]models.Artist, int, error) {
	// Implement pagination similar to FindAllArtworks
	return nil, 0, errors.New("not implemented")
}

func (r *Repository) FindArtistByID(ctx context.Context, artistID int64) (*models.Artist, error) {
	// Implement similar to FindArtworkByID
	return nil, errors.New("not implemented")
}

func (r *Repository) FindAllCategories(ctx context.Context) ([]string, error) {
	categories := []string{}
	query := `SELECT DISTINCT category FROM artworks WHERE category IS NOT NULL AND category != '' ORDER BY category ASC`
	rows, err := r.executor.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository.FindAllCategories: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			return nil, fmt.Errorf("repository.FindAllCategories.Scan: %w", err)
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

func (r *Repository) CheckFavorites(ctx context.Context, userID string, artworkIDs []int64) (map[int64]bool, error) {
	favoriteMap := make(map[int64]bool)
	if userID == "" || len(artworkIDs) == 0 {
		return favoriteMap, nil
	}
	query := `SELECT artwork_id FROM user_favorite_artworks WHERE user_id = $1 AND artwork_id = ANY($2)`
	rows, err := r.executor.Query(ctx, query, userID, artworkIDs)
	if err != nil {
		return nil, fmt.Errorf("repository.CheckFavorites: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var artworkID int64
		if err := rows.Scan(&artworkID); err != nil {
			return nil, fmt.Errorf("repository.CheckFavorites.Scan: %w", err)
		}
		favoriteMap[artworkID] = true
	}
	return favoriteMap, nil
}

func (r *Repository) AddFavorite(ctx context.Context, userID string, artworkID int64) error {
	query := `INSERT INTO user_favorite_artworks (user_id, artwork_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.executor.Exec(ctx, query, userID, artworkID)
	if err != nil {
		return fmt.Errorf("repository.AddFavorite: %w", err)
	}
	return nil
}

func (r *Repository) RemoveFavorite(ctx context.Context, userID string, artworkID int64) error {
	query := `DELETE FROM user_favorite_artworks WHERE user_id = $1 AND artwork_id = $2`
	cmdTag, err := r.executor.Exec(ctx, query, userID, artworkID)
	if err != nil {
		return fmt.Errorf("repository.RemoveFavorite: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		// This isn't necessarily an error, could just mean it wasn't a favorite to begin with.
		// Returning ErrNotFound could be misleading. Returning nil is often fine.
	}
	return nil
}
