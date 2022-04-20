package api

// SheetOrganizer is the API message for a sheet organizer.
type SheetOrganizer struct {
	ID int `jsonapi:"primary,sheetOrganizer"`

	// Related fields
	SheetID     int  `jsonapi:"attr,sheetId"`
	PrincipalID int  `jsonapi:"attr,principalId"`
	Starred     bool `jsonapi:"attr,starred"`
	Pinned      bool `jsonapi:"attr,pinned"`
}

// SheetOrganizerUpsert is the message to upsert a sheet organizer.
// NOTE: We use PATCH for Upsert, this is inspired by https://google.aip.dev/134#patch-and-put
type SheetOrganizerUpsert struct {
	SheetID     int
	PrincipalID int
	Starred     *bool `jsonapi:"attr,starred"`
	Pinned      *bool `jsonapi:"attr,pinned"`
}

// SheetOrganizerFind is the API message to find sheet organizers.
type SheetOrganizerFind struct {
	SheetID     *int
	PrincipalID *int
	Starred     *bool
	Pinned      *bool
}
