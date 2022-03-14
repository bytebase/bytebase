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
func (s *BookmarkService) CreateBookmark(ctx context.Context, create *api.BookmarkCreate) (*api.BookmarkRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	bookmark, err := createBookmark(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return bookmark, nil
}

// FindBookmarkList retrieves a list of bookmarks based on find.
func (s *BookmarkService) FindBookmarkList(ctx context.Context, find *api.BookmarkFind) ([]*api.BookmarkRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findBookmarkList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// FindBookmark retrieves a single bookmark based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *BookmarkService) FindBookmark(ctx context.Context, find *api.BookmarkFind) (*api.BookmarkRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	bookmarkRawList, err := findBookmarkList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(bookmarkRawList) == 0 {
		return nil, nil
	} else if len(bookmarkRawList) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d activities with filter %+v, expect 1. ", len(bookmarkRawList), find)}
	}
	return bookmarkRawList[0], nil
}

// DeleteBookmark deletes an existing bookmark by ID.
// Returns ENOTFOUND if bookmark does not exist.
func (s *BookmarkService) DeleteBookmark(ctx context.Context, delete *api.BookmarkDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteBookmark(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createBookmark creates a new bookmark.
func createBookmark(ctx context.Context, tx *sql.Tx, create *api.BookmarkCreate) (*api.BookmarkRaw, error) {
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
	var bookmarkRaw api.BookmarkRaw
	if err := row.Scan(
		&bookmarkRaw.ID,
		&bookmarkRaw.CreatorID,
		&bookmarkRaw.CreatedTs,
		&bookmarkRaw.UpdaterID,
		&bookmarkRaw.UpdatedTs,
		&bookmarkRaw.Name,
		&bookmarkRaw.Link,
	); err != nil {
		return nil, FormatError(err)
	}

	return &bookmarkRaw, nil
}

func findBookmarkList(ctx context.Context, tx *sql.Tx, find *api.BookmarkFind) ([]*api.BookmarkRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("creator_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
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

	// Iterate over result set and deserialize rows into bookmarkRawList.
	var bookmarkRawList []*api.BookmarkRaw
	for rows.Next() {
		var bookmark api.BookmarkRaw
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

		bookmarkRawList = append(bookmarkRawList, &bookmark)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return bookmarkRawList, nil
}

// deleteBookmark permanently deletes a bookmark by ID.
func deleteBookmark(ctx context.Context, tx *sql.Tx, delete *api.BookmarkDelete) error {
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
