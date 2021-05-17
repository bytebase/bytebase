package api

import "context"

type Bookmark struct {
	ID int `jsonapi:"primary,bookmark"`

	// Standard fields
	CreatorId   int
	Creator     *Principal `jsonapi:"attr,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"attr,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
	Link string `jsonapi:"attr,link"`
}

type BookmarkCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
	Link string `jsonapi:"attr,link"`
}

type BookmarkFind struct {
	ID *int

	// Standard fields
	CreatorId   *int
	WorkspaceId *int
}

type BookmarkDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

type BookmarkService interface {
	CreateBookmark(ctx context.Context, create *BookmarkCreate) (*Bookmark, error)
	FindBookmarkList(ctx context.Context, find *BookmarkFind) ([]*Bookmark, error)
	FindBookmark(ctx context.Context, find *BookmarkFind) (*Bookmark, error)
	DeleteBookmark(ctx context.Context, delete *BookmarkDelete) error
}
