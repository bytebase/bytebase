package v1

import (
	"github.com/bytebase/bytebase/plugin/db"
)

// IssueCreate is the API message for creating an issue.
type IssueCreate struct {
	// Related fields
	ProjectID   string `json:"projectID"`
	Database    string `json:"database"`
	Environment string `json:"environment"`

	// Domain specific fields
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	MigrationType db.MigrationType `json:"migrationType"`
	Statement     string           `json:"statement"`
	SchemaVersion string           `json:"schemaVersion"`
}
