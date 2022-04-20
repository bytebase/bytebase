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

// SheetOrganizerFind is the API message to find a sheet organizer.
type SheetOrganizerFind struct {
	SheetID     int
	PrincipalID int
}

// SheetOrganizerUpsert is the API message to upsert a sheet organizer.
type SheetOrganizerUpsert struct {
	SheetID     int
	PrincipalID int
	Starred     *bool `jsonapi:"attr,starred"`
	Pinned      *bool `jsonapi:"attr,pinned"`
}
