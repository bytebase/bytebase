package api

const (
	// EnvironmentLabelKey is the reserved key for environment.
	EnvironmentLabelKey string = "bb.environment"
	// TenantLabelKey is the label key for tenant.
	TenantLabelKey = "bb.tenant"
	// LocationLabelKey is the label key for location.
	LocationLabelKey = "bb.location"

	// DatabaseLabelSizeMax is the maximum size of database labels.
	DatabaseLabelSizeMax = 4
	labelLengthMax       = 63
)

// DatabaseLabel is the label associated with a database.
type DatabaseLabel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// DatabaseLabelFind finds the labels associated with the database.
type DatabaseLabelFind struct {
	// Standard fields
	RowStatus *RowStatus

	// Related fields
	DatabaseID int
}

// DatabaseLabelUpsert upserts the label associated with the database.
type DatabaseLabelUpsert struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int
	RowStatus RowStatus

	// Related fields
	DatabaseID int
	Key        string

	// Domain specific fields
	Value string
}
