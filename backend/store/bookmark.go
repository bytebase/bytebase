package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
)

var (
	_ api.BookmarkService = (*BookmarkService)(nil)
)

// BookmarkService represents a service for managing bookmark.
type BookmarkService struct {
	l  *bytebase.Logger
	db *DB
}

// NewBookmarkService returns a new instance of BookmarkService.
func NewBookmarkService(logger *bytebase.Logger, db *DB) *BookmarkService {
	return &BookmarkService{l: logger, db: db}
}

// CreateBookmark creates a new bookmark.
func (s *BookmarkService) CreateBookmark(ctx context.Context, create *api.BookmarkCreate) (*api.Bookmark, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	bookmark, err := createBookmark(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
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
	defer tx.Rollback()

	list, err := findBookmarkList(ctx, tx, find)
	if err != nil {
		return []*api.Bookmark{}, err
	}

	return list, nil
}

// FindBookmark retrieves a single bookmark based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *BookmarkService) FindBookmark(ctx context.Context, find *api.BookmarkFind) (*api.Bookmark, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findBookmarkList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("bookmark not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warnf("found %d activities with filter %v, expect 1. ", len(list), find)
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
	defer tx.Rollback()

	err = deleteBookmark(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createBookmark creates a new bookmark.
func createBookmark(ctx context.Context, tx *Tx, create *api.BookmarkCreate) (*api.Bookmark, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO bookmark (
			creator_id,
			updater_id,
			workspace_id,
			name,
			link
		)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, workspace_id, name, link
	`,
		create.CreatorId,
		create.CreatorId,
		create.WorkspaceId,
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
		&bookmark.CreatorId,
		&bookmark.CreatedTs,
		&bookmark.UpdaterId,
		&bookmark.UpdatedTs,
		&bookmark.WorkspaceId,
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
	if v := find.CreatorId; v != nil {
		where, args = append(where, "creator_id = ?"), append(args, *v)
	}
	if v := find.WorkspaceId; v != nil {
		where, args = append(where, "workspace_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			workspace_id,
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
			&bookmark.CreatorId,
			&bookmark.CreatedTs,
			&bookmark.UpdaterId,
			&bookmark.UpdatedTs,
			&bookmark.WorkspaceId,
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
func deleteBookmark(ctx context.Context, tx *Tx, delete *api.BookmarkDelete) error {
	// Remove row from database.
	result, err := tx.ExecContext(ctx, `DELETE FROM bookmark WHERE id = ?`, delete.ID)
	if err != nil {
		return FormatError(err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("bookmark ID not found: %d", delete.ID)}
	}

	return nil
}
