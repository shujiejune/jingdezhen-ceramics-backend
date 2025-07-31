package forum

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"time"

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

	FindAllPosts(ctx context.Context, filters models.PostFilters) ([]models.ForumPost, int, error)
	FindPostByID(ctx context.Context, postID int64) (*models.ForumPost, error)
	SearchPosts(ctx context.Context, query string, page, limit int) ([]models.ForumPost, int, error)
	FindCommentByID(ctx context.Context, commentID int64) (*models.ForumComment, error)
	FindCommentsByPostID(ctx context.Context, postID int64) ([]models.ForumComment, error)
	FindAllCategories(ctx context.Context) ([]models.ForumCategory, error)
	FindCategoryByID(ctx context.Context, categoryID int64) (*models.ForumCategory, error)
	FindAllTags(ctx context.Context) ([]models.Tag, error)

	CheckPostOwnership(ctx context.Context, postID int64, userID string) (bool, error)
	CheckCommentOwnership(ctx context.Context, commentID int64, userID string) (bool, error)
	CreatePost(ctx context.Context, userID string, data models.CreatePostRequest) (*models.ForumPost, error)
	UpdatePost(ctx context.Context, postID int64, data models.UpdatePostRequest) (*models.ForumPost, error)
	DeletePost(ctx context.Context, postID int64) error
	CreateComment(ctx context.Context, userID string, postID int64, parentCommentID *int64, content string) (*models.ForumComment, error)
	UpdateComment(ctx context.Context, commentID int64, content string) (*models.ForumComment, error)
	DeleteComment(ctx context.Context, commentID int64) error

	IsPostLikedByUser(ctx context.Context, userID string, postID int64) (bool, error)
	IsPostSavedByUser(ctx context.Context, userID string, postID int64) (bool, error)
	IsCommentLikedByUser(ctx context.Context, userID string, commentID int64) (bool, error)
	AddLikeToPost(ctx context.Context, userID string, postID int64) (int, error)
	RemoveLikeFromPost(ctx context.Context, userID string, postID int64) (int, error)
	AddSaveToPost(ctx context.Context, userID string, postID int64) error
	RemoveSaveFromPost(ctx context.Context, userID string, postID int64) error
	AddLikeToComment(ctx context.Context, userID string, commentID int64) (int, error)
	RemoveLikeFromComment(ctx context.Context, userID string, commentID int64) (int, error)
	CheckPostsLikedByUser(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error)
	CheckPostsSavedByUser(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error)
	CheckCommentsLikedByUser(ctx context.Context, userID string, commentIDs []int64) (map[int64]bool, error)
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

func (r *Repository) scanForumPostWithoutContent(row Scannable) (*models.ForumPost, error) {
	var post models.ForumPost
	var avatarUrl sql.NullString

	err := row.Scan(
		&post.ID,
		&post.UserID,
		&post.AuthorNickname,
		&avatarUrl,
		&post.CategoryID,
		&post.CategoryName,
		&post.Title,
		&post.IsPinned,
		&post.ViewCount,
		&post.LikeCount,
		&post.CommentCount,
		&post.LastActivityAt,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.Tags,
	)
	if err != nil {
		return nil, err
	}
	if avatarUrl.Valid {
		post.AuthorAvatarURL = &avatarUrl.String
	} else {
		post.AuthorAvatarURL = nil
	}
	return &post, nil
}

// FindAllPosts retrieves all the forum posts with filters and pagination
func (r *Repository) FindAllPosts(ctx context.Context, filters models.PostFilters) ([]models.ForumPost, int, error) {
	var args []any
	argIdx := 1

	baseQuery := `
		FROM forum_posts fp
		JOIN users u ON fp.user_id = u.id
		LEFT JOIN forum_categories fc ON fp.category_id = fc.id
	`
	if filters.Tag != "" {
		baseQuery += ` JOIN forum_post_tags fpt ON fp.id = fpt.post_id
		               JOIN tags t ON fpt.tag_id = t.id`
	}

	whereClause := "WHERE fp.is_archived = FALSE"
	if filters.CategoryID > 0 {
		whereClause += fmt.Sprintf(" AND fp.category_id = $%d", argIdx)
		args = append(args, filters.CategoryID)
		argIdx++
	}
	if filters.Tag != "" {
		whereClause += fmt.Sprintf(" AND t.name = $%d", argIdx)
		args = append(args, filters.Tag)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(DISTINCT fp.id) " + baseQuery + whereClause
	if err := r.executor.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository.FindAllPosts.Count: %w", err)
	}
	if total == 0 {
		return []models.ForumPost{}, 0, nil
	}

	orderByClause := "ORDER BY fp.is_pinned DESC, fp.last_activity_at DESC"
	if filters.Sort == "hottest" {
		// A simple "hottest" algorithm: likes + comments, weighted by time decay.
		orderByClause = "ORDER BY fp.is_pinned DESC, (fp.like_count + fp.comment_count) / POW(EXTRACT(EPOCH FROM (NOW() - fp.created_at))/3600, 1.8) DESC"
	} else if filters.Sort == "most recently published" {
		orderByClause = "ORDER BY fp.is_pinned DESC, fp.created_at DESC"
	} else if filters.Sort == "most recently updated" {
		orderByClause = "ORDER BY fp.is_pinned DESC, fp.updated_at DESC"
	}

	selectQuery := `
		SELECT DISTINCT fp.id, fp.user_id, u.nickname as author_nickname, u.avatar_url as author_avatar_url,
		       fp.category_id, fc.name as category_name, fp.title, fp.is_pinned,
		       fp.view_count, fp.comment_count, fp.like_count,
		       fp.last_activity_at, fp.created_at, fp.updated_at
	`
	limitOffsetClause := fmt.Sprintf(" %s LIMIT $%d OFFSET $%d", orderByClause, argIdx, argIdx+1)
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	fullQuery := selectQuery + baseQuery + whereClause + limitOffsetClause
	rows, err := r.executor.Query(ctx, fullQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.FindAllPosts.Query: %w", err)
	}
	defer rows.Close()

	posts := []models.ForumPost{}
	for rows.Next() {
		post, err := r.scanForumPostWithoutContent(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository.FindAllPosts.Scan: %w", err)
		}
		posts = append(posts, *post)
	}
	return posts, total, nil
}

// FindPostByID retrieves a single, detailed forum post.
func (r *Repository) FindPostByID(ctx context.Context, postID int64) (*models.ForumPost, error) {
	var post models.ForumPost
	query := `
		SELECT 
			fp.id, fp.user_id, u.nickname as author_nickname, u.avatar_url as author_avatar_url,
			fp.category_id, fc.name as category_name, fp.title, fp.content, fp.is_pinned,
			fp.view_count, 
			(SELECT COUNT(*) FROM forum_comments WHERE post_id = fp.id) as comment_count,
			(SELECT COUNT(*) FROM forum_post_likes WHERE post_id = fp.id) as like_count,
			fp.last_activity_at, fp.created_at, fp.updated_at
		FROM forum_posts fp
		JOIN users u ON fp.user_id = u.id
		LEFT JOIN forum_categories fc ON fp.category_id = fc.id
		WHERE fp.id = $1
	`
	err := r.executor.QueryRow(ctx, query, postID).Scan(
		&post.ID, &post.UserID, &post.AuthorNickname, &post.AuthorAvatarURL,
		&post.CategoryID, &post.CategoryName, &post.Title, &post.Content, &post.IsPinned,
		&post.ViewCount, &post.CommentCount, &post.LikeCount,
		&post.LastActivityAt, &post.CreatedAt, &post.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindPostByID: %w", err)
	}
	return &post, nil
}

func (r *Repository) SearchPosts(ctx context.Context, query string, page, limit int) ([]models.ForumPost, int, error) {
	// Use websearch_to_tsquery for more natural language queries (e.g., handles "and", "or")
	tsQuery := "websearch_to_tsquery('simple', $1)"

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM forum_posts WHERE search_vector @@ %s", tsQuery)
	if err := r.executor.QueryRow(ctx, countQuery, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository.SearchPosts.Count: %w", err)
	}
	if total == 0 {
		return []models.ForumPost{}, 0, nil
	}

	offset := (page - 1) * limit
	// ts_rank_cd calculates relevance score. We order by relevance.
	searchQuery := fmt.Sprintf(`
		SELECT fp.id, fp.user_id, u.nickname as author_nickname, u.avatar_url as author_avatar_url,
		       fp.category_id, fc.name as category_name, fp.title, fp.is_pinned,
		       fp.view_count, fp.comment_count, fp.like_count,
		       fp.last_activity_at, fp.created_at, fp.updated_at
		FROM forum_posts fp
		JOIN users u ON fp.user_id = u.id
		LEFT JOIN forum_categories fc ON fp.category_id = fc.id
		WHERE fp.search_vector @@ %s
		ORDER BY ts_rank_cd(fp.search_vector, %s) DESC
		LIMIT $2 OFFSET $3
	`, tsQuery, tsQuery)

	rows, err := r.executor.Query(ctx, searchQuery, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.SearchPosts.Query: %w", err)
	}
	defer rows.Close()

	posts := []models.ForumPost{}
	for rows.Next() {
		post, err := r.scanForumPostWithoutContent(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository.SearchPosts.Scan: %w", err)
		}
		posts = append(posts, *post)
	}
	return posts, total, nil
}

func (r *Repository) FindAllCategories(ctx context.Context) ([]models.ForumCategory, error) {
	categories := []models.ForumCategory{}
	query := `SELECT id, name, description FROM forum_categories ORDER BY display_order ASC`
	rows, err := r.executor.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository.FindAllCategories: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cat models.ForumCategory
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Description); err != nil {
			return nil, fmt.Errorf("repository.FindAllCategories.Scan: %w", err)
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

func (r *Repository) FindCategoryByID(ctx context.Context, categoryID int64) (*models.ForumCategory, error) {
	cat := models.ForumCategory{}
	query := `
		SELECT id, name, description, display_order
		FROM forum_categories
		WHERE id = $1
	`
	err := r.executor.QueryRow(ctx, query, categoryID).Scan(
		&cat.ID, &cat.Name, &cat.Description, &cat.DisplayOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.FindCategoryByID: %w", err)
	}

	return &cat, nil
}

func (r *Repository) FindAllTags(ctx context.Context) ([]models.Tag, error) {
	tags := []models.Tag{}
	// This query also gets the count of posts for each tag, useful for a tag cloud.
	query := `
		SELECT t.id, t.name, COUNT(fpt.post_id) as post_count
		FROM tags t
		JOIN forum_post_tags fpt ON t.id = fpt.tag_id
		GROUP BY t.id, t.name
		ORDER BY post_count DESC, t.name ASC
	`
	rows, err := r.executor.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository.FindAllTags: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.PostCount); err != nil {
			return nil, fmt.Errorf("repository.FindAllTags.Scan: %w", err)
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (r *Repository) scanComment(row Scannable) (*models.ForumComment, error) {
	var cmt models.ForumComment
	var avatarUrl sql.NullString
	var parentID sql.NullInt64

	err := row.Scan(
		&cmt.ID,
		&cmt.PostID,
		&cmt.UserID,
		&cmt.AuthorNickname,
		&avatarUrl,
		&parentID,
		&cmt.Content,
		&cmt.LikeCount,
		&cmt.CreatedAt,
		&cmt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if avatarUrl.Valid {
		cmt.AuthorAvatarURL = &avatarUrl.String
	} else {
		cmt.AuthorAvatarURL = nil
	}
	if parentID.Valid {
		cmt.ParentCommentID = &parentID.Int64
	} else {
		cmt.ParentCommentID = nil
	}
	return &cmt, nil
}

func (r *Repository) FindCommentByID(ctx context.Context, commentID int64) (*models.ForumComment, error) {
	var comment models.ForumComment
	query := `
		SELECT id, post_id, user_id, u.nickname as author_nickname, u.avatar_url as author_avatar_url,
		parent_comment_id, content, created_at, updated_at, 
		FROM forum_comments fc
		JOIN users u ON fc.user_id = u.id
		LEFT JOIN forum_comment_likes fcl ON fc.user_id = fcl.user_id
		WHERE id = $1
	`
	err := r.executor.QueryRow(ctx, query, commentID).Scan(&comment.ID, &comment.PostID, &comment)
	if err != nil {
		return nil, fmt.Errorf("repository.FindCommentByID: %w", err)
	}
	return &comment, nil
}

// FindCommentsByPostID retrieves all comments for a post, ordered for threading.
func (r *Repository) FindCommentsByPostID(ctx context.Context, postID int64) ([]models.ForumComment, error) {
	comments := []models.ForumComment{}
	// This recursive CTE builds a path for each comment, allowing us to sort them
	// so that replies always appear after their parents.
	query := `
		WITH RECURSIVE comment_thread AS (
			SELECT 
				id, post_id, user_id, parent_comment_id, content, created_at, updated_at,
				ARRAY[created_at, id] AS path
			FROM forum_comments
			WHERE parent_comment_id IS NULL AND post_id = $1
			UNION ALL
			SELECT 
				c.id, c.post_id, c.user_id, c.parent_comment_id, c.content, c.created_at, c.updated_at,
				ct.path || ARRAY[c.created_at, c.id]
			FROM forum_comments c
			JOIN comment_thread ct ON c.parent_comment_id = ct.id
		)
		SELECT 
			ct.id, ct.post_id, ct.user_id, u.nickname as author_nickname, u.avatar_url as author_avatar_url,
			ct.parent_comment_id, ct.content,
			(SELECT COUNT(*) FROM forum_comment_likes WHERE comment_id = ct.id) as like_count,
			ct.created_at, ct.updated_at
		FROM comment_thread ct
		JOIN users u ON ct.user_id = u.id
		ORDER BY ct.path;
	`
	rows, err := r.executor.Query(ctx, query, postID)
	if err != nil {
		return nil, fmt.Errorf("repository.FindCommentsByPostID: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var c models.ForumComment
		if err := rows.Scan(
			&c.ID, &c.PostID, &c.UserID, &c.AuthorNickname, &c.AuthorAvatarURL,
			&c.ParentCommentID, &c.Content, &c.LikeCount, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository.FindCommentsByPostID.Scan: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func (r *Repository) CheckPostOwnership(ctx context.Context, postID int64, userID string) (bool, error)

func (r *Repository) CheckCommentOwnership(ctx context.Context, commentID int64, userID string) (bool, error)

func (r *Repository) CreatePost(ctx context.Context, userID string, data models.CreatePostRequest) (*models.ForumPost, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Insert the main post
	postQuery := `
		INSERT INTO forum_posts (user_id, category_id, title, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at, last_activity_at
	`
	var postID int64
	var createdAt, updatedAt, lastActivityAt time.Time
	err = tx.QueryRow(ctx, postQuery, userID, data.CategoryID, data.Title, data.Content).Scan(&postID, &createdAt, &updatedAt, &lastActivityAt)
	if err != nil {
		return nil, fmt.Errorf("repository.CreatePost.InsertPost: %w", err)
	}

	// 2. Handle tags
	if len(data.Tags) > 0 {
		// This is a complex operation: find existing tags, create new ones, then link them.
		// For brevity, a full implementation is omitted, but it would involve more queries within the transaction.
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// 3. Fetch the full post to return it
	return r.FindPostByID(ctx, postID)
}

func (r *Repository) UpdatePost(ctx context.Context, postID int64, data models.UpdatePostRequest) (*models.ForumPost, error)

func (r *Repository) DeletePost(ctx context.Context, postID int64) error

func (r *Repository) CreateComment(ctx context.Context, userID string, postID int64, parentCommentID *int64, content string) (*models.ForumComment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Insert the comment
	commentQuery := `
		INSERT INTO forum_comments (user_id, post_id, parent_comment_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	var comment models.ForumComment
	comment.UserID = userID
	comment.PostID = postID
	comment.ParentCommentID = parentCommentID
	comment.Content = content

	err = tx.QueryRow(ctx, commentQuery, userID, postID, parentCommentID, content).Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository.CreateComment.Insert: %w", err)
	}

	// 2. Update the post's comment count and last activity time
	updatePostQuery := `
		UPDATE forum_posts 
		SET comment_count = comment_count + 1, last_activity_at = NOW() 
		WHERE id = $1
	`
	if _, err := tx.Exec(ctx, updatePostQuery, postID); err != nil {
		return nil, fmt.Errorf("repository.CreateComment.UpdatePost: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Fetch author details to populate the struct fully before returning
	// ...
	return &comment, nil
}

func (r *Repository) UpdateComment(ctx context.Context, commentID int64, content string) (*models.ForumComment, error)

func (r *Repository) DeleteComment(ctx context.Context, commentID int64) error

func (r *Repository) IsPostLikedByUser(ctx context.Context, userID string, postID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM forum_post_likes WHERE user_id = $1 AND post_id = $2)`
	err := r.executor.QueryRow(ctx, query, userID, postID).Scan(&exists)
	return exists, err
}

func (r *Repository) IsPostSavedByUser(ctx context.Context, userID string, postID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM forum_post_saves WHERE user_id = $1 AND post_id = $2)`
	err := r.executor.QueryRow(ctx, query, userID, postID).Scan(&exists)
	return exists, err
}

func (r *Repository) IsCommentLikedByUser(ctx context.Context, userID string, commentID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM forum_comment_likes WHERE user_id = $1 AND comment_id = $2)`
	err := r.executor.QueryRow(ctx, query, userID, commentID).Scan(&exists)
	return exists, err
}

func (r *Repository) AddLikeToPost(ctx context.Context, userID string, postID int64) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "INSERT INTO forum_post_likes (user_id, post_id) VALUES ($1, $2)", userID, postID)
	if err != nil {
		return 0, err
	}

	var newCount int
	err = tx.QueryRow(ctx, "UPDATE forum_posts SET like_count = like_count + 1 WHERE id = $1 RETURNING like_count", postID).Scan(&newCount)
	if err != nil {
		return 0, err
	}

	return newCount, tx.Commit(ctx)
}

func (r *Repository) RemoveLikeFromPost(ctx context.Context, userID string, postID int64) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM forum_post_likes WHERE user_id = $1 AND post_id = $2", userID, postID)
	if err != nil {
		return 0, err
	}

	var newCount int
	err = tx.QueryRow(ctx, "UPDATE forum_posts SET like_count = GREATEST(0, like_count - 1) WHERE id = $1 RETURNING like_count", postID).Scan(&newCount)
	if err != nil {
		return 0, err
	}

	return newCount, tx.Commit(ctx)
}

func (r *Repository) AddSaveToPost(ctx context.Context, userID string, postID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "INSERT INTO forum_post_saves (user_id, post_id) VALUES ($1, $2)", userID, postID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) RemoveSaveFromPost(ctx context.Context, userID string, postID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM forum_post_saves WHERE user_id = $1 AND post_id = $2", userID, postID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) AddLikeToComment(ctx context.Context, userID string, commentID int64) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "INSERT INTO forum_comment_likes (user_id, comment_id) VALUES ($1, $2)", userID, commentID)
	if err != nil {
		return 0, err
	}

	var newCount int
	err = tx.QueryRow(ctx, "UPDATE forum_comments SET like_count = like_count + 1 WHERE id = $1 RETURNING like_count", commentID).Scan(&newCount)
	if err != nil {
		return 0, err
	}

	return newCount, tx.Commit(ctx)
}

func (r *Repository) RemoveLikeFromComment(ctx context.Context, userID string, commentID int64) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM forum_comment_likes WHERE user_id = $1 AND comment_id = $2", userID, commentID)
	if err != nil {
		return 0, err
	}

	var newCount int
	err = tx.QueryRow(ctx, "UPDATE forum_comments SET like_count = GREATEST(0, like_count - 1) WHERE id = $1 RETURNING like_count", commentID).Scan(&newCount)
	if err != nil {
		return 0, err
	}

	return newCount, tx.Commit(ctx)
}

func (r *Repository) CheckPostsLikedByUser(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error) {
	likedMap := make(map[int64]bool)
	if userID == "" || len(postIDs) == 0 {
		return likedMap, nil
	}
	query := `SELECT post_id FROM forum_post_likes WHERE user_id = $1 AND post_id = ANY($2)`
	rows, err := r.executor.Query(ctx, query, userID, postIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		if err := rows.Scan(&postID); err != nil {
			return nil, err
		}
		likedMap[postID] = true
	}
	return likedMap, nil
}

func (r *Repository) CheckPostsSavedByUser(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error) {
	savedMap := make(map[int64]bool)
	if userID == "" || len(postIDs) == 0 {
		return savedMap, nil
	}
	query := `SELECT post_id FROM forum_post_saves WHERE user_id = $1 AND post_id = ANY($2)`
	rows, err := r.executor.Query(ctx, query, userID, postIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		if err := rows.Scan(&postID); err != nil {
			return nil, err
		}
		savedMap[postID] = true
	}
	return savedMap, nil
}

func (r *Repository) CheckCommentsLikedByUser(ctx context.Context, userID string, commentIDs []int64) (map[int64]bool, error) {
	likedMap := make(map[int64]bool)
	if userID == "" || len(commentIDs) == 0 {
		return likedMap, nil
	}
	query := `SELECT comment_id FROM forum_comment_likes WHERE user_id = $1 AND comment_id = ANY($2)`
	rows, err := r.executor.Query(ctx, query, userID, commentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var commentID int64
		if err := rows.Scan(&commentID); err != nil {
			return nil, err
		}
		likedMap[commentID] = true
	}
	return likedMap, nil
}
