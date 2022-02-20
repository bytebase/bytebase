package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.BookmarkService = (*BookmarkService)(nil)
)

// BookmarkService represents a service for managing bookmark.
type BookmarkService struct {
	l  *zap.Logger
	db *DB
}

// NewBookmarkService returns a new instance of BookmarkService.
func NewBookmarkService(logger *zap.Logger, db *DB) *BookmarkService {
	return &BookmarkService{l: logger, db: db}
}

// CreateBookmark creates a new bookmark.
func (s *BookmarkService) CreateBookmark(ctx context.Context, create *api.BookmarkCreate) (*api.Bookmark, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	bookmark, err := pgCreateBookmark(ctx, tx.Tx, create)
	if err != nil {
		return nil, err
	}
	if _, err := createBookmark(ctx, tx.PTx, create); err != nil {
		return nil, err
	}

	if err := tx.Tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return bookmark, nil
}

// FindBookmarkList retrieves a list of bookmarks based on find.
func (s *BookmarkService) FindBookmarkList(ctx context.Context, find *api.BookmarkFind) ([]*api.Bookmark, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	list, err := findBookmarkList(ctx, tx, find)
	if err != nil {
		return []*api.Bookmark{}, err
	}

	return list, nil
}

// FindBookmark retrieves a single bookmark based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *BookmarkService) FindBookmark(ctx context.Context, find *api.BookmarkFind) (*api.Bookmark, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	list, err := findBookmarkList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d activities with filter %+v, expect 1. ", len(list), find)}
	}
	return list[0], nil
}

// DeleteBookmark deletes an existing bookmark by ID.
// Returns ENOTFOUND if bookmark does not exist.
func (s *BookmarkService) DeleteBookmark(ctx context.Context, delete *api.BookmarkDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	if err := pgDeleteBookmark(ctx, tx.Tx, delete); err != nil {
		return FormatError(err)
	}
	if err := deleteBookmark(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.Tx.Commit(); err != nil {
		return FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createBookmark creates a new bookmark.
func createBookmark(ctx context.Context, tx *sql.Tx, create *api.BookmarkCreate) (*api.Bookmark, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO bookmark (
			creator_id,
			updater_id,
			name,
			link
		)
		VALUES (?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, link
	`,
		create.CreatorID,
		create.CreatorID,
		create.Name,
		create.Link,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var bookmark api.Bookmark
	if err := row.Scan(
		&bookmark.ID,
		&bookmark.CreatorID,
		&bookmark.CreatedTs,
		&bookmark.UpdaterID,
		&bookmark.UpdatedTs,
		&bookmark.Name,
		&bookmark.Link,
	); err != nil {
		return nil, FormatError(err)
	}

	return &bookmark, nil
}

// pgCreateBookmark creates a new bookmark.
func pgCreateBookmark(ctx context.Context, tx *sql.Tx, create *api.BookmarkCreate) (*api.Bookmark, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO bookmark (
			creator_id,
			updater_id,
			name,
			link
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, link
	`,
		create.CreatorID,
		create.CreatorID,
		create.Name,
		create.Link,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var bookmark api.Bookmark
	if err := row.Scan(
		&bookmark.ID,
		&bookmark.CreatorID,
		&bookmark.CreatedTs,
		&bookmark.UpdaterID,
		&bookmark.UpdatedTs,
		&bookmark.Name,
		&bookmark.Link,
	); err != nil {
		return nil, FormatError(err)
	}

	return &bookmark, nil
}

func findBookmarkList(ctx context.Context, tx *Tx, find *api.BookmarkFind) (_ []*api.Bookmark, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, "creator_id = ?"), append(args, *v)
	}

	rows, err := tx.Tx.QueryContext(ctx, `
		SELECT
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
		    name,
		    link
		FROM bookmark
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Bookmark, 0)
	for rows.Next() {
		var bookmark api.Bookmark
		if err := rows.Scan(
			&bookmark.ID,
			&bookmark.CreatorID,
			&bookmark.CreatedTs,
			&bookmark.UpdaterID,
			&bookmark.UpdatedTs,
			&bookmark.Name,
			&bookmark.Link,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &bookmark)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// deleteBookmark permanently deletes a bookmark by ID.
func deleteBookmark(ctx context.Context, tx *sql.Tx, delete *api.BookmarkDelete) error {
	// Remove row from database.
	result, err := tx.ExecContext(ctx, `DELETE FROM bookmark WHERE id = ?`, delete.ID)
	if err != nil {
		return FormatError(err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &common.Error{Code: common.NotFound, Err: fmt.Errorf("bookmark ID not found: %d", delete.ID)}
	}

	return nil
}

// pgDeleteBookmark permanently deletes a bookmark by ID.
func pgDeleteBookmark(ctx context.Context, tx *sql.Tx, delete *api.BookmarkDelete) error {
	// Remove row from database.
	result, err := tx.ExecContext(ctx, `DELETE FROM bookmark WHERE id = $1`, delete.ID)
	if err != nil {
		return FormatError(err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &common.Error{Code: common.NotFound, Err: fmt.Errorf("bookmark ID not found: %d", delete.ID)}
	}

	return nil
}
