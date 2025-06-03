package user

import (
	"context"
	"database/sql" // For sql.ErrNoRows
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	// "github.com/Masterminds/squirrel" // Optional: for SQL query building
)

// RepositoryInterface defines methods for interacting with user storage.
type RepositoryInterface interface {
	FindByID(ctx context.Context, userID string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, user *models.User, passwordHash string) (*models.User, error) // Assuming you might add direct user creation
	Update(ctx context.Context, userID string, updateData models.UserUpdateData) (*models.User, error)
	ListAll(ctx context.Context, page, limit int) ([]models.User, int, error) // For admin: list users
	UpdateRole(ctx context.Context, userID string, newRole string) error      // For admin: update role

	// User Notes specific methods
	GetUserNoteByID(ctx context.Context, noteID int, userID string) (*models.UserNote, error)
	ListUserNotes(ctx context.Context, userID string, page, limit int) ([]models.UserNote, int, error)
	CreateUserNote(ctx context.Context, userID string, data models.CreateUserNoteData) (*models.UserNote, error)
	UpdateUserNote(ctx context.Context, noteID int, userID string, data models.UpdateUserNoteData) (*models.UserNote, error)
	DeleteUserNote(ctx context.Context, noteID int, userID string) error
	MarkNoteAsPublished(ctx context.Context, noteID int, forumPostID int) error
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) RepositoryInterface {
	return &Repository{db: db}
}

func (r *Repository) FindByID(ctx context.Context, userID string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, nickname, email, role, avatar_url, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.Nickname, &user.Email, &user.Role, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows || strings.Contains(err.Error(), "no rows in result set") { // pgx might return different error
			return nil, models.ErrNotFound // Define this error in models
		}
		return nil, fmt.Errorf("repository.FindByID: %w", err)
	}
	return user, nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	// Similar to FindByID, but queries by email
	// Important for checking if email exists during signup if you implement it
	user := &models.User{}
	query := `SELECT id, nickname, email, role, avatar_url, password_hash, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Nickname, &user.Email, &user.Role, &user.AvatarURL, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows || strings.Contains(err.Error(), "no rows in result set") {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindByEmail: %w", err)
	}
	return user, nil
}

func (r *Repository) Create(ctx context.Context, user *models.User, passwordHash string) (*models.User, error) {
	// This would be for direct email/password signup if Supabase isn't handling ALL user creation
	query := `
        INSERT INTO users (nickname, email, password_hash, role, avatar_url, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, created_at, updated_at`
	err := r.db.QueryRow(ctx, query,
		user.Nickname, user.Email, passwordHash, user.Role, user.AvatarURL, time.Now(),
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Handle potential duplicate email error (unique constraint)
		return nil, fmt.Errorf("repository.CreateUser: %w", err)
	}
	return user, nil
}

func (r *Repository) Update(ctx context.Context, userID string, data models.UserUpdateData) (*models.User, error) {
	// Build query dynamically based on fields provided in UserUpdateData
	// For simplicity, let's assume nickname and avatar_url are updatable
	var setClauses []string
	var args []interface{}
	argIdx := 1

	if data.Nickname != nil {
		setClauses = append(setClauses, fmt.Sprintf("nickname = $%d", argIdx))
		args = append(args, *data.Nickname)
		argIdx++
	}
	if data.AvatarURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar_url = $%d", argIdx))
		args = append(args, *data.AvatarURL)
		argIdx++
	}

	if len(setClauses) == 0 {
		return r.FindByID(ctx, userID) // No fields to update, return current user
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, userID) // For WHERE clause

	query := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d
	                     RETURNING id, nickname, email, role, avatar_url, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx)

	updatedUser := &models.User{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&updatedUser.ID, &updatedUser.Nickname, &updatedUser.Email, &updatedUser.Role, &updatedUser.AvatarURL, &updatedUser.CreatedAt, &updatedUser.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.UpdateUser: %w", err)
	}
	return updatedUser, nil
}

// --- Admin specific methods ---
func (r *Repository) ListAll(ctx context.Context, page, limit int) ([]models.User, int, error) {
	offset := (page - 1) * limit
	query := `SELECT id, nickname, email, role, avatar_url, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListAllUsers: %w", err)
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Nickname, &user.Email, &user.Role, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("repository.ListAllUsers.Scan: %w", err)
		}
		users = append(users, user)
	}

	var total int
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListAllUsers.Count: %w", err)
	}

	return users, total, nil
}

func (r *Repository) UpdateRole(ctx context.Context, userID string, newRole string) error {
	query := `UPDATE users SET role = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, newRole, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("repository.UpdateUserRole: %w", err)
	}
	return nil
}

// --- User Notes Methods ---
func (r *Repository) GetUserNoteByID(ctx context.Context, noteID int, userID string) (*models.UserNote, error) {
	note := &models.UserNote{}
	query := `SELECT id, user_id, title, content, entity_type, entity_id, is_published_to_forum, forum_post_id, created_at, updated_at
	          FROM user_notes WHERE id = $1 AND user_id = $2`
	err := r.db.QueryRow(ctx, query, noteID, userID).Scan(
		&note.ID, &note.UserID, &note.Title, &note.Content, &note.EntityType, &note.EntityID,
		&note.IsPublishedToForum, &note.ForumPostID, &note.CreatedAt, &note.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows || strings.Contains(err.Error(), "no rows") {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetUserNoteByID: %w", err)
	}
	return note, nil
}

func (r *Repository) ListUserNotes(ctx context.Context, userID string, page, limit int) ([]models.UserNote, int, error) {
	// Implement pagination similar to ListAllUsers
	notes := []models.UserNote{}
	offset := (page - 1) * limit
	query := `SELECT id, user_id, title, entity_type, entity_id, is_published_to_forum, created_at, updated_at
	          FROM user_notes WHERE user_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListUserNotes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var note models.UserNote
		// Scan fewer fields for list view if full content not needed
		if err := rows.Scan(&note.ID, &note.UserID, &note.Title, &note.EntityType, &note.EntityID, &note.IsPublishedToForum, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("repository.ListUserNotes.Scan: %w", err)
		}
		notes = append(notes, note)
	}

	var total int
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM user_notes WHERE user_id = $1", userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListUserNotes.Count: %w", err)
	}
	return notes, total, nil
}

func (r *Repository) CreateUserNote(ctx context.Context, userID string, data models.CreateUserNoteData) (*models.UserNote, error) {
	note := models.UserNote{
		UserID:     userID,
		Title:      data.Title,
		Content:    data.Content,
		EntityType: data.EntityType,
		EntityID:   data.EntityID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	query := `INSERT INTO user_notes (user_id, title, content, entity_type, entity_id, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	err := r.db.QueryRow(ctx, query, note.UserID, note.Title, note.Content, note.EntityType, note.EntityID, note.CreatedAt, note.UpdatedAt).Scan(&note.ID)
	if err != nil {
		return nil, fmt.Errorf("repository.CreateUserNote: %w", err)
	}
	return &note, nil
}

func (r *Repository) UpdateUserNote(ctx context.Context, noteID int, userID string, data models.UpdateUserNoteData) (*models.UserNote, error) {
	// Similar dynamic query building as r.Update for users
	// Ensure to check ownership: WHERE id = $X AND user_id = $Y
	// For brevity, a full dynamic implementation is omitted here.
	// Placeholder:
	currentNote, err := r.GetUserNoteByID(ctx, noteID, userID)
	if err != nil {
		return nil, err
	}
	if data.Title != nil {
		currentNote.Title = *data.Title
	}
	if data.Content != nil {
		currentNote.Content = *data.Content
	}
	currentNote.UpdatedAt = time.Now()

	query := `UPDATE user_notes SET title = $1, content = $2, updated_at = $3
              WHERE id = $4 AND user_id = $5
              RETURNING id, user_id, title, content, entity_type, entity_id, is_published_to_forum, forum_post_id, created_at, updated_at`
	err = r.db.QueryRow(ctx, query, currentNote.Title, currentNote.Content, currentNote.UpdatedAt, noteID, userID).Scan(
		&currentNote.ID, &currentNote.UserID, &currentNote.Title, &currentNote.Content, &currentNote.EntityType, &currentNote.EntityID,
		&currentNote.IsPublishedToForum, &currentNote.ForumPostID, &currentNote.CreatedAt, &currentNote.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.UpdateUserNote: %w", err)
	}
	return currentNote, nil
}

func (r *Repository) DeleteUserNote(ctx context.Context, noteID int, userID string) error {
	query := `DELETE FROM user_notes WHERE id = $1 AND user_id = $2`
	cmdTag, err := r.db.Exec(ctx, query, noteID, userID)
	if err != nil {
		return fmt.Errorf("repository.DeleteUserNote: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return models.ErrNotFound // Or ErrForbidden if you prefer that for ownership failures
	}
	return nil
}

func (r *Repository) MarkNoteAsPublished(ctx context.Context, noteID int, forumPostID int) error {
	query := `UPDATE user_notes SET is_published_to_forum = TRUE, forum_post_id = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, forumPostID, time.Now(), noteID)
	if err != nil {
		return fmt.Errorf("repository.MarkNoteAsPublished: %w", err)
	}
	return nil
}
