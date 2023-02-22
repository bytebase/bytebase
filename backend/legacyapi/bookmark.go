package api

// Bookmark is the API message for a bookmark.
type Bookmark struct {
	ID int `jsonapi:"primary,bookmark"`

	// Standard fields
	CreatorID int        `jsonapi:"attr,creatorID"`
	Creator   *Principal `jsonapi:"relation,creator"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
	Link string `jsonapi:"attr,link"`
}

// BookmarkCreate is the API message for creating a bookmark.
type BookmarkCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
	Link string `jsonapi:"attr,link"`
}
