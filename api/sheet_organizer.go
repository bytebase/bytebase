package api

// SheetOrganizer is the API message for a sheet organizer.
type SheetOrganizer struct {
	ID          int  `jsonapi:"primary,sheetOrganizer"`
	SheetID     int  `jsonapi:"attr,sheetId"`
	PrincipalID int  `jsonapi:"attr,principalId"`
	Starred     bool `jsonapi:"attr,starred"`
	Pinned      bool `jsonapi:"attr,pinned"`
}

// SheetOrganizerCreate is the API message for creating a sheet organizer.
type SheetOrganizerCreate struct {
	SheetID     int
	PrincipalID int
	Starred     *bool `jsonapi:"attr,starred"`
	Pinned      *bool `jsonapi:"attr,pinned"`
}

// SheetOrganizerFind is the API message for finding sheet organizers.
type SheetOrganizerFind struct {
	SheetID     *int
	PrincipalID *int
	Starred     *bool
	Pinned      *bool
}

// SheetOrganizerPatch is the API message for patching a sheet organizer.
type SheetOrganizerPatch struct {
	ID          int
	SheetID     int
	PrincipalID int
	Starred     *bool `jsonapi:"attr,starred"`
	Pinned      *bool `jsonapi:"attr,pinned"`
}
