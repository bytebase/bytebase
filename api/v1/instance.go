package v1

import (
	"github.com/bytebase/bytebase/plugin/db"
)

// Instance is the API message for an instance.
type Instance struct {
	ID int `json:"id"`

	// Related fields
	Environment    string        `json:"environment"`
	DataSourceList []*DataSource `json:"dataSourceList"`

	// Domain specific fields
	Name          string  `json:"name"`
	Engine        db.Type `json:"engine"`
	EngineVersion string  `json:"engineVersion"`
	ExternalLink  string  `json:"externalLink"`
	Host          string  `json:"host"`
	Port          string  `json:"port"`
	Database      string  `json:"database"`
}

// InstanceCreate is the API message for creating an instance.
type InstanceCreate struct {
	// Related fields
	Environment    string              `json:"environment"`
	DataSourceList []*DataSourceCreate `json:"dataSourceList"`

	// Domain specific fields
	Name         string  `json:"name"`
	Engine       db.Type `json:"engine"`
	ExternalLink string  `json:"externalLink"`
	Host         string  `json:"host"`
	Port         string  `json:"port"`
	Database     string  `json:"database"`
}

// InstancePatch is the API message for patching an instance.
type InstancePatch struct {
	// Related fields
	DataSourceList []*DataSourceCreate `json:"dataSourceList"`

	// Domain specific fields
	Name         *string `json:"name"`
	ExternalLink *string `json:"externalLink"`
	Host         *string `json:"host"`
	Port         *string `json:"port"`
	Database     *string `json:"database"`
}

// InstanceDatabasePatch is the API message for patching an instance database.
type InstanceDatabasePatch struct {
	// Project is the project resource ID.
	Project *string `json:"project"`
}
