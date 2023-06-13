package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// BookmarkMessage is the message for bookmark.
type BookmarkMessage struct {
	// Link is the link of the bookmark.
	Link string

	// Name is the name of the bookmark.
	Name string

	// Output only fields.
	//
	// UID is the unique ID of the bookmark.
	UID int
	// CreatorUID is the unique ID of the creator.
	CreatorUID int
}

// DeleteBookmarkMessage is the message to delete a bookmark.
type DeleteBookmarkMessage struct {
	// UID is the unique ID of the bookmark.
	UID int

	// CreatorUID is the unique ID of the creator.
	CreatorUID int
}

// CreateBookmarkV2 creates a new bookmark.
func (s *Store) CreateBookmarkV2(ctx context.Context, create *BookmarkMessage, principleUID int) (*BookmarkMessage, error) {
	query := `
		INSERT INTO bookmark (
			creator_id,
			updater_id,
			name,
			link
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, creator_id, link
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var bookmark BookmarkMessage
	if err := tx.QueryRowContext(ctx, query,
		principleUID,
		principleUID,
		create.Name,
		create.Link,
	).Scan(
		&bookmark.UID,
		&bookmark.Name,
		&bookmark.CreatorUID,
		&bookmark.Link,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, errors.Wrapf(err, "failed to query row")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return &bookmark, nil
}

// ListBookmarkV2 lists all bookmarks.
func (s *Store) ListBookmarkV2(ctx context.Context, principalUID int) ([]*BookmarkMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	bookmarks, err := s.listBookmarkImplV2(ctx, tx, principalUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list bookmarks")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return bookmarks, nil
}

// DeleteBookmarkV2 try to delete a bookmark.
func (s *Store) DeleteBookmarkV2(ctx context.Context, delete *DeleteBookmarkMessage) error {
	var args []any
	var where []string
	where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, delete.UID)
	where, args = append(where, fmt.Sprintf("creator_id = $%d", len(args)+1)), append(args, delete.CreatorUID)

	query := fmt.Sprintf(`
		DELETE FROM bookmark WHERE %s
	`, strings.Join(where, " AND "))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrapf(err, "failed to delete bookmark %v", delete)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &common.Error{Code: common.NotFound, Err: errors.Errorf("bookmark not found: %v", delete)}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}

	return nil
}

func (*Store) listBookmarkImplV2(ctx context.Context, tx *Tx, creatorUID int) ([]*BookmarkMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	where, args = append(where, fmt.Sprintf("creator_id = $%d", len(args)+1)), append(args, creatorUID)

	query := fmt.Sprintf(`
		SELECT
			id,
			creator_id,
			name,
			link
		FROM bookmark WHERE %s
	`, strings.Join(where, " AND "))

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query bookmark for user %v", creatorUID)
	}
	defer rows.Close()

	var bookmarks []*BookmarkMessage

	for rows.Next() {
		var bookmark BookmarkMessage
		if err := rows.Scan(
			&bookmark.UID,
			&bookmark.CreatorUID,
			&bookmark.Name,
			&bookmark.Link,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan bookmark for user %v", creatorUID)
		}

		bookmarks = append(bookmarks, &bookmark)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate bookmark for user %v", creatorUID)
	}

	return bookmarks, nil
}
