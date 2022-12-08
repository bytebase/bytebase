package v1

import "github.com/bytebase/bytebase/api"

// DataSource is the API message for a data source.
type DataSource struct {
	ID int `json:"id"`

	// Related fields
	DatabaseID int `json:"databaseId"`

	// Domain specific fields
	Name     string             `json:"name"`
	Type     api.DataSourceType `json:"type"`
	Username string             `json:"username"`

	// HostOverride and PortOverride are only used for read-only data sources for user's read-replica instances.
	HostOverride string `json:"hostOverride"`
	PortOverride string `json:"portOverride"`
}

// DataSourceCreate is the API message for creating a data source.
type DataSourceCreate struct {
	// Domain specific fields
	Name         string             `json:"name"`
	Type         api.DataSourceType `json:"type"`
	Username     string             `json:"username"`
	Password     string             `json:"password"`
	SslCa        string             `json:"sslCa"`
	SslCert      string             `json:"sslCert"`
	SslKey       string             `json:"sslKey"`
	HostOverride string             `json:"hostOverride"`
	PortOverride string             `json:"portOverride"`
}
