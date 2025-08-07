package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RepositoryInterface defines methods for interacting with user storage.
type RepositoryInterface interface {
	FindByID(ctx context.Context, userID string) (*models.User, error)
	FindByIDs(ctx context.Context, userIDs []string) ([]models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByNickname(ctx context.Context, nickname string) (*models.User, error)
	FindByPasswordResetToken(ctx context.Context, token string) (*models.User, error)

	SetPasswordResetToken(ctx context.Context, userID string, token string, expiresAt time.Time) error
	UpdatePasswordAndClearResetToken(ctx context.Context, userID string, passwordHash string) error
	UpdateActivationToken(ctx context.Context, userID, newToken string, expiresAt time.Time) error

	CreateInactiveUser(ctx context.Context, user *models.User, passwordHash, activationToken string, expiresAt time.Time) (*models.User, error)
	ActivateUser(ctx context.Context, token string) (*models.User, error)
	CreateOAuthUser(ctx context.Context, user *models.User) (*models.User, error) // Assuming you might add direct user creation
	Update(ctx context.Context, userID string, updateData models.UserUpdateData) (*models.User, error)

	FindEnrolledCoursesByUserID(ctx context.Context, userID string) ([]models.EnrolledCourseResponse, error)

	ListAll(ctx context.Context, page, limit int) ([]models.User, int, error) // For admin: list users
	UpdateRole(ctx context.Context, userID string, newRole string) error      // For admin: update role
}

// This interface represents anything that can execute a SQL query,
// which includes both a connection pool and a transaction.
type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	db       *pgxpool.Pool
	executor DBExecutor
}

func NewRepository(db *pgxpool.Pool) RepositoryInterface {
	return &Repository{
		db:       db,
		executor: db,
	}
}

// BeginTx starts a new database transaction.
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// WithTx returns a new instance of the Repository that is "scoped" to the provided transaction.
// All database operations on the returned repository will be part of this single transaction.
func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{
		db:       r.db,
		executor: tx, // The executor is now the transaction, not the pool
	}
}

func (r *Repository) scanUser(row pgx.Row) (*models.User, error) {
	var user models.User
	var avatarURL sql.NullString

	err := row.Scan(
		&user.ID,
		&user.Nickname,
		&user.Email,
		&user.Role,
		&avatarURL,
		&user.ProfileData,
		&user.ProfileData,
		&user.AuthProvider,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	} else {
		user.AvatarURL = nil
	}

	return &user, nil
}

func (r *Repository) scanUserWithPasswordHash(row pgx.Row) (*models.User, error) {
	var user models.User
	var passwordHash sql.NullString
	var avatarURL sql.NullString

	err := row.Scan(
		&user.ID,
		&user.Nickname,
		&user.Email,
		&passwordHash,
		&user.Role,
		&avatarURL,
		&user.ProfileData,
		&user.AuthProvider,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if passwordHash.Valid {
		user.PasswordHash = &passwordHash.String
	} else {
		user.PasswordHash = nil
	}

	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	} else {
		user.AvatarURL = nil
	}

	return &user, nil
}

func (r *Repository) FindByID(ctx context.Context, userID string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, nickname, email, role, avatar_url, profile_data, auth_provider, is_active, created_at, updated_at FROM users WHERE id = $1`

	row := r.executor.QueryRow(ctx, query, userID)
	user, err := r.scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindByID: %w", err)
	}
	return user, nil
}

// Used in notification service to prevent the N+1 query problem.
func (r *Repository) FindByIDs(ctx context.Context, userIDs []string) ([]models.User, error) {
	if len(userIDs) == 0 {
		return []models.User{}, nil // Return empty slice if no IDs are provided
	}

	// Use the PostgreSQL ANY() operator for a clean and secure "IN" clause.
	query := `SELECT * FROM users WHERE id = ANY($1)`

	rows, err := r.db.Query(ctx, query, userIDs)
	if err != nil {
		return nil, fmt.Errorf("query for users by ids failed: %w", err)
	}

	// pgx.CollectRows scans all resulting rows into a slice of structs.
	users, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.User])
	if err != nil {
		return nil, fmt.Errorf("failed to collect user rows: %w", err)
	}

	return users, nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	// Similar to FindByID, but queries by email
	// Important for checking if email exists during signup if you implement it
	user := &models.User{}
	query := `SELECT id, nickname, email, password_hash, role, avatar_url, profile_data, auth_provider, is_active, created_at, updated_at FROM users WHERE email = $1`

	row := r.executor.QueryRow(ctx, query, email)
	user, err := r.scanUserWithPasswordHash(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindByEmail: %w", err)
	}
	return user, nil
}

func (r *Repository) FindByNickname(ctx context.Context, nickname string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, nickname, email, role, avatar_url, profile_data, auth_provider, is_active, created_at, updated_at FROM users WHERE nickname = $1`

	row := r.executor.QueryRow(ctx, query, nickname)
	user, err := r.scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindByNickname: %w", err)
	}
	return user, nil
}

func (r *Repository) FindByPasswordResetToken(ctx context.Context, token string) (*models.User, error) {
	user := &models.User{}

	query := `
	SELECT id, nickname, email, password_hash, role, avatar_url, profile_data, auth_provider, auth_provider_id, is_active, created_at, updated_at
	FROM users
	WHERE password_reset_token = $1 AND password_reset_expires_at > NOW()
	`

	row := r.executor.QueryRow(ctx, query, token)
	user, err := r.scanUserWithPasswordHash(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrInvalidToken
		}
		return nil, fmt.Errorf("repository.FindUserByPasswordResetToken: %w", err)
	}
	return user, nil
}

func (r *Repository) SetPasswordResetToken(ctx context.Context, userID string, token string, expiresAt time.Time) error {
	log.Printf("DATABASE: Saving reset token [%s] for user [%s]", token, userID)
	query := `
	UPDATE users
	SET password_reset_token = $1, password_reset_expires_at = $2, updated_at = NOW()
	WHERE id = $3
	`
	cmdTag, err := r.db.Exec(ctx, query, token, expiresAt, userID)
	if err != nil {
		return fmt.Errorf("repository.SetPasswordResetToken: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return models.ErrNotFound // userID not found, no update to password_reset_token
	}

	return nil
}

func (r *Repository) UpdatePasswordAndClearResetToken(ctx context.Context, userID string, passwordHash string) error {
	query := `
	UPDATE users
	SET password_hash = $1, password_reset_token = $2, updated_at = NOW()
	WHERE id = $3
	`
	cmdTag, err := r.db.Exec(ctx, query, passwordHash, "", userID)
	if err != nil {
		return fmt.Errorf("repository.UpdatePasswordAndClearResetToken: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return models.ErrNotFound // userID not found, no update to password_reset_token
	}

	return nil
}

func (r *Repository) UpdateActivationToken(ctx context.Context, userID, newToken string, expiresAt time.Time) error {
	query := `
	UPDATE users
	SET activation_token = $1, activation_token_expires_at = $2, updated_at = NOW()
	WHERE id = $3
	`
	cmdTag, err := r.db.Exec(ctx, query, newToken, expiresAt, userID)
	if err != nil {
		return fmt.Errorf("repository.UpdateActivationToken: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return models.ErrNotFound // userID not found, no update to password_reset_token
	}

	return nil
}

// Specifically for the email/password signup flow
func (r *Repository) CreateInactiveUser(ctx context.Context, user *models.User, passwordHash, activationToken string, expiresAt time.Time) (*models.User, error) {
	query := `
        INSERT INTO users (nickname, email, password_hash, role, activation_token, activation_token_expires_at, auth_provider)
        VALUES ($1, $2, $3, $4, $5, $6, 'email')
        RETURNING id, created_at, updated_at`
	err := r.db.QueryRow(ctx, query,
		user.Nickname, user.Email, passwordHash, user.Role, activationToken, expiresAt,
	).Scan(&user.ID, &user.IsActive, &user.AuthProvider, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository.CreateInactiveUser: %w", err)
	}
	return user, err
}

func (r *Repository) ActivateUser(ctx context.Context, token string) (*models.User, error) {
	// Find user by token, set is_active = true, and clear the token
	user := &models.User{}
	query := `
        UPDATE users
        SET is_active = TRUE, activation_token = NULL, activation_token_expires_at = NULL, updated_at = NOW()
        WHERE activation_token = $1 AND activation_token_expires_at > NOW() AND is_active = FALSE
        RETURNING id, nickname, email, role, avatar_url, profile_data, auth_provider, is_active, created_at, updated_at`

	row := r.executor.QueryRow(ctx, query, token)
	user, err := r.scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrInvalidToken
		}
		return nil, fmt.Errorf("repository.ActivateUser: %w", err)
	}
	return user, nil
}

// Specifically for OAuth signup flow (Google/WeChat)
func (r *Repository) CreateOAuthUser(ctx context.Context, user *models.User) (*models.User, error) {
	query := `
        INSERT INTO users (nickname, email, role, auth_provider, auth_provider_id, is_active)
        VALUES ($1, $2, $3, $4, $5, TRUE)
        RETURNING id, created_at, updated_at`
	err := r.db.QueryRow(ctx, query,
		user.Nickname, user.Email, user.Role, user.AuthProvider, user.AuthProviderID,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Handle potential duplicate email error (unique constraint)
		return nil, fmt.Errorf("repository.CreateOAuthUser: %w", err)
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
	if data.OtherContact != nil {
		setClauses = append(setClauses, fmt.Sprintf("profile_data = jsonb_set(COALESCE(profile_data, '{}'::jsonb), '{other_contact}', $%d::jsonb)", argIdx))
		args = append(args, *data.OtherContact)
		argIdx++
	}

	if len(setClauses) == 0 {
		return r.FindByID(ctx, userID) // No fields to update, return current user
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, userID) // For WHERE clause

	query := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d RETURNING id, nickname, email, role, avatar_url, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx)

	updatedUser := &models.User{}
	row := r.executor.QueryRow(ctx, query, args...)
	updatedUser, err := r.scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("repository.UpdateUser: %w", err)
	}
	return updatedUser, nil
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

// --- Admin specific methods ---
func (r *Repository) ListAll(ctx context.Context, page, limit int) ([]models.User, int, error) {
	offset := (page - 1) * limit
	query := `SELECT id, nickname, email, password_hash, role, avatar_url, profile_data, auth_provider, auth_provider_id, is_active, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListAllUsers: %w", err)
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID, &user.Nickname, &user.Email, &user.PasswordHash, &user.Role, &user.AvatarURL, &user.ProfileData, &user.AuthProvider, &user.AuthProviderID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
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
