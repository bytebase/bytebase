package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// bookmarkRaw is the store model for an Bookmark.
// Fields have exactly the same meanings as Bookmark.
type bookmarkRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	Name string
	Link string
}

// toBookmark creates an instance of Bookmark based on the bookmarkRaw.
// This is intended to be called when we need to compose an Bookmark relationship.
func (raw *bookmarkRaw) toBookmark() *api.Bookmark {
	return &api.Bookmark{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Domain specific fields
		Name: raw.Name,
		Link: raw.Link,
	}
}

// CreateBookmark creates an instance of Bookmark
func (s *Store) CreateBookmark(ctx context.Context, create *api.BookmarkCreate) (*api.Bookmark, error) {
	bookmarkRaw, err := s.createBookmarkRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bookmark with BookmarkCreate[%+v], error[%w]", create, err)
	}
	bookmark, err := s.composeBookmark(ctx, bookmarkRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Bookmark with bookmarkRaw[%+v], error[%w]", bookmarkRaw, err)
	}
	return bookmark, nil
}

// GetBookmarkByID gets an instance of Bookmark
func (s *Store) GetBookmarkByID(ctx context.Context, id int) (*api.Bookmark, error) {
	find := &api.BookmarkFind{ID: &id}
	bookmarkRaw, err := s.getBookmarkRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get Bookmark with ID[%d], error[%w]", id, err)
	}
	if bookmarkRaw == nil {
		return nil, nil
	}
	bookmark, err := s.composeBookmark(ctx, bookmarkRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Bookmark with bookmarkRaw[%+v], error[%w]", bookmarkRaw, err)
	}
	return bookmark, nil
}

// FindBookmark finds a list of Bookmark instances
func (s *Store) FindBookmark(ctx context.Context, find *api.BookmarkFind) ([]*api.Bookmark, error) {
	bookmarkRawList, err := s.findBookmarkRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find Bookmark list with BookmarkFind[%+v], error[%w]", find, err)
	}
	var bookmarkList []*api.Bookmark
	for _, raw := range bookmarkRawList {
		bookmark, err := s.composeBookmark(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose Bookmark with bookmarkRaw[%+v], error[%w]", raw, err)
		}
		bookmarkList = append(bookmarkList, bookmark)
	}
	return bookmarkList, nil
}

//
// private function
//

func (s *Store) composeBookmark(ctx context.Context, raw *bookmarkRaw) (*api.Bookmark, error) {
	bookmark := raw.toBookmark()

	creator, err := s.GetPrincipalByID(ctx, bookmark.CreatorID)
	if err != nil {
		return nil, err
	}
	bookmark.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, bookmark.UpdaterID)
	if err != nil {
		return nil, err
	}
	bookmark.Updater = updater

	return bookmark, nil
}

// createBookmarkRaw creates a new bookmark.
func (s *Store) createBookmarkRaw(ctx context.Context, create *api.BookmarkCreate) (*bookmarkRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	bookmark, err := createBookmarkImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return bookmark, nil
}

// findBookmarkRaw retrieves a list of bookmarks based on find.
func (s *Store) findBookmarkRaw(ctx context.Context, find *api.BookmarkFind) ([]*bookmarkRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findBookmarkImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getBookmarkRaw retrieves a single bookmark based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getBookmarkRaw(ctx context.Context, find *api.BookmarkFind) (*bookmarkRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	bookmarkRawList, err := findBookmarkImpl(ctx, tx.PTx, find)
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
func (s *Store) DeleteBookmark(ctx context.Context, delete *api.BookmarkDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteBookmarkImpl(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createBookmarkImpl creates a new bookmark.
func createBookmarkImpl(ctx context.Context, tx *sql.Tx, create *api.BookmarkCreate) (*bookmarkRaw, error) {
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
	var bookmarkRaw bookmarkRaw
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

func findBookmarkImpl(ctx context.Context, tx *sql.Tx, find *api.BookmarkFind) ([]*bookmarkRaw, error) {
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
	var bookmarkRawList []*bookmarkRaw
	for rows.Next() {
		var bookmark bookmarkRaw
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

// deleteBookmarkImpl permanently deletes a bookmark by ID.
func deleteBookmarkImpl(ctx context.Context, tx *sql.Tx, delete *api.BookmarkDelete) error {
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
