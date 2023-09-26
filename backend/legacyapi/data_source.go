package api

// DataSourceType is the type of data source.
type DataSourceType string

const (
	// Admin is the ADMIN type of data source.
	Admin DataSourceType = "ADMIN"
	// RO is the read-only type of data source.
	RO DataSourceType = "RO"
)
