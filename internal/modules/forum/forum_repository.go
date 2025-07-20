package forum

import (
	"context"
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
	FindAllPosts(ctx context.Context, filters models.PostFilters) ([]models.ForumPost, int, error)
	SearchPosts(ctx context.Context, query string, page, limit int) ([]models.ForumPost, int, error)
	FindPostByID(ctx context.Context, postID int64) (*models.ForumPost, error)
	FindCommentsByPostID(ctx context.Context, postID int64) ([]models.ForumComment, error)
	FindAllCategories(ctx context.Context) ([]models.ForumCategory, error)
	FindAllTags(ctx context.Context) ([]models.Tag, error)
	CreatePost(ctx context.Context, userID string, data models.CreatePostRequest) (*models.ForumPost, error)
	UpdatePost(ctx context.Context, postID int64, data models.UpdatePostRequest) (*models.ForumPost, error)
	DeletePost(ctx context.Context, postID int64) error
	CreateComment(ctx context.Context, userID string, postID int64, parentCommentID *int64, content string) (*models.ForumComment, error)
	UpdateComment(ctx context.Context, commentID int64, content string) (*models.ForumComment, error)
	DeleteComment(ctx context.Context, commentID int64) error
	AddLikeToPost(ctx context.Context, userID string, postID int64) error
	RemoveLikeFromPost(ctx context.Context, userID string, postID int64) error
	AddSaveForPost(ctx context.Context, userID string, postID int64) error
	RemoveSaveFromPost(ctx context.Context, userID string, postID int64) error
	AddLikeToComment(ctx context.Context, userID string, commentID int64) error
	RemoveLikeFromComment(ctx context.Context, userID string, commentID int64) error
	CheckPostOwnership(ctx context.Context, postID int64, userID string) (bool, error)
	CheckCommentOwnership(ctx context.Context, commentID int64, userID string) (bool, error)
	// ... transaction methods
}

type Repository struct {
	db       *pgxpool.Pool
	executor DBExecutor
}

func NewRepository(db *pgxpool.Pool) RepositoryInterface {
	return &Repository{db: db, executor: db}
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

// ... other repository methods (FindAllPosts, Search, Create, Update, Delete, Likes, Saves, etc.)
// These would be quite long, so I'll omit the full text but the patterns would be similar
// to other modules (e.g., using dynamic query building for FindAllPosts filters).
