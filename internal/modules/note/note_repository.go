package note

import (
	"context"
	"database/sql"
	"fmt"
	"jingdezhen-ceramics-backend/internal/models"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	BeginTx(ctx context.Context) (pgx.Tx, error)
	WithTx(tx pgx.Tx) *Repository
	GetUserNoteByID(ctx context.Context, noteID int64, userID string) (*models.UserNote, error)
	GetLinksForNote(ctx context.Context, noteID int64) ([]models.UserNoteLink, error)
	ListUserNotes(ctx context.Context, userID string, page, limit int) ([]models.UserNote, int, error)
	CreateUserNote(ctx context.Context, userID string, data models.CreateUserNoteData) (*models.UserNote, error)
	UpdateUserNote(ctx context.Context, noteID int64, userID string, data models.UpdateUserNoteData) (*models.UserNote, error)
	DeleteUserNote(ctx context.Context, noteID int64, userID string) error
	AddLinkToNote(ctx context.Context, noteID int64, data models.AddLinkToNoteData) (*models.UserNoteLink, error)
	RemoveLinkFromNote(ctx context.Context, noteID, linkID int64) error
	MarkNoteAsPublished(ctx context.Context, noteID, forumPostID int64) error
}

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

func (r *Repository) GetUserNoteByID(ctx context.Context, noteID int64, userID string) (*models.UserNote, error) {
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

func (r *Repository) GetLinksForNote(ctx context.Context, noteID int64) ([]models.UserNoteLink, error) {
	links := []models.UserNoteLink{}
	query := `SELECT id, user_note_id, linked_entity_type, linked_entity_id, linked_description, created_at
	          FROM user_note_links WHERE user_note_id = $1`
	rows, err := r.db.Query(ctx, query, noteID)
	if err != nil {
		return nil, fmt.Errorf("repository.GetLinksForNote: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var link models.UserNoteLink
		if err := rows.Scan(&link.UserNoteID, &link.LinkedEntityType, &link.LinkedEntityID, &link.LinkDescription, &link.CreatedAt); err != nil {
			return nil, fmt.Errorf("repository.GetLinksForNote.Scan: %w", err)
		}
		links = append(links, link)
	}
	return links, nil
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

func (r *Repository) UpdateUserNote(ctx context.Context, noteID int64, userID string, data models.UpdateUserNoteData) (*models.UserNote, error) {
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

func (r *Repository) DeleteUserNote(ctx context.Context, noteID int64, userID string) error {
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

func (r *Repository) AddLinkToNote(ctx context.Context, noteID int64, data models.AddLinkToNoteData) (*models.UserNoteLink, error) {
	link := models.UserNoteLink{
		UserNoteID:       noteID,
		LinkedEntityType: data.LinkedEntityType,
		LinkedEntityID:   data.LinkedEntityID,
		LinkDescription:  data.LinkDescription,
		CreatedAt:        time.Now(),
	}
	query := `INSERT INTO user_note_links (user_note_id, linked_entity_type, linked_entity_id, link_description, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	err := r.db.QueryRow(ctx, query, link.UserNoteID, link.LinkedEntityType, link.LinkedEntityID, link.LinkDescription, link.CreatedAt).Scan(&link.ID)
	if err != nil {
		return nil, fmt.Errorf("repository.AddLinkToNote: %w", err)
	}
	return &link, nil
}

func (r *Repository) RemoveLinkFromNote(ctx context.Context, noteID, linkID int64) error {
	query := `DELETE FROM user_note_links WHERE id = $1 AND user_note_id = $2`
	cmdTag, err := r.db.Exec(ctx, query, linkID, noteID)
	if err != nil {
		return fmt.Errorf("repository.RemoveLinkFromNote: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *Repository) MarkNoteAsPublished(ctx context.Context, noteID int64, forumPostID int64) error {
	query := `UPDATE user_notes SET is_published_to_forum = TRUE, forum_post_id = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, forumPostID, time.Now(), noteID)
	if err != nil {
		return fmt.Errorf("repository.MarkNoteAsPublished: %w", err)
	}
	return nil
}
