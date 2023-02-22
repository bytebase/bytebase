package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// BookmarkMessage is the message for bookmark.
type BookmarkMessage struct {
	// Link is the link of the bookmark.
	Link string

	// Name is the name of the bookmark.
	Name string

	// Output only fields.
	//
	// ID is the unique ID of the bookmark.
	ID int
	// CreatorUID is the unique ID of the creator.
	CreatorUID int
}

// listBookmarkMessage is the message for listing bookmarks.
type listBookmarkMessage struct {
	// id is the unique ID of the bookmark.
	id *int
	// creatorUID is the unique ID of the creator.
	creatorUID *int
}

// ToAPIBookmark converts a BookmarkMessage to an API Bookmark.
func (b *BookmarkMessage) ToAPIBookmark() *api.Bookmark {
	return &api.Bookmark{
		ID:        b.ID,
		CreatorID: b.CreatorUID,
		Name:      b.Name,
		Link:      b.Link,
	}
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
		&bookmark.ID,
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

	bookmarks, err := s.listBookmarkImplV2(ctx, tx, &listBookmarkMessage{creatorUID: &principalUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list bookmarks")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return bookmarks, nil
}

// GetBookmarkV2 gets a bookmark by ID.
func (s *Store) GetBookmarkV2(ctx context.Context, bookmarkUID int) (*BookmarkMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	bookmarks, err := s.listBookmarkImplV2(ctx, tx, &listBookmarkMessage{id: &bookmarkUID})
	if err != nil {
		return nil, err
	}
	if len(bookmarks) == 0 {
		return nil, nil
	}
	if len(bookmarks) > 1 {
		return nil, errors.Errorf("find %d bookmarks for bookmark ID %d", len(bookmarks), bookmarkUID)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	return bookmarks[0], nil
}

// DeleteBookmarkV2 try to delete a bookmark.
func (s *Store) DeleteBookmarkV2(ctx context.Context, bookmarkUID int) error {
	var args []interface{}
	var where []string
	where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, bookmarkUID)

	query := fmt.Sprintf(`
		DELETE FROM bookmark WHERE %s
	`, strings.Join(where, " AND "))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, query, bookmarkUID)
	if err != nil {
		return errors.Wrapf(err, "failed to delete bookmark of principal id %d", bookmarkUID)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &common.Error{Code: common.NotFound, Err: errors.Errorf("bookmark ID not found: %d", bookmarkUID)}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}

	return nil
}

func (s *Store) listBookmarkImplV2(ctx context.Context, tx *Tx, list *listBookmarkMessage) ([]*BookmarkMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	if v := list.creatorUID; v != nil {
		where, args = append(where, fmt.Sprintf("creator_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := list.id; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}

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
		return nil, errors.Wrapf(err, "failed to query bookmark of %v", list)
	}
	defer rows.Close()

	var bookmarks []*BookmarkMessage

	for rows.Next() {
		var bookmark BookmarkMessage
		if err := rows.Scan(
			&bookmark.ID,
			&bookmark.CreatorUID,
			&bookmark.Name,
			&bookmark.Link,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan bookmark of %v", list)
		}

		bookmarks = append(bookmarks, &bookmark)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate bookmark of %v", list)
	}

	return bookmarks, nil
}
