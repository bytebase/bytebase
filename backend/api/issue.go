package api

import (
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
)

// IssueStatus is the status of an issue.
type IssueStatus string

const (
	// IssueOpen is the issue status for OPEN.
	IssueOpen IssueStatus = "OPEN"
	// IssueDone is the issue status for DONE.
	IssueDone IssueStatus = "DONE"
	// IssueCanceled is the issue status for CANCELED.
	IssueCanceled IssueStatus = "CANCELED"
)

// IssueType is the type of an issue.
type IssueType string

const (
	// IssueGeneral is the issue type for general issues.
	IssueGeneral IssueType = "bb.issue.general"
	// IssueDatabaseCreate is the issue type for creating databases.
	IssueDatabaseCreate IssueType = "bb.issue.database.create"
	// IssueDatabaseSchemaUpdate is the issue type for updating database schemas (DDL).
	IssueDatabaseSchemaUpdate IssueType = "bb.issue.database.schema.update"
	// IssueDatabaseSchemaUpdateGhost is the issue type for updating database schemas using gh-ost.
	IssueDatabaseSchemaUpdateGhost IssueType = "bb.issue.database.schema.update.ghost"
	// IssueDatabaseDataUpdate is the issue type for updating database data (DML).
	IssueDatabaseDataUpdate IssueType = "bb.issue.database.data.update"
	// IssueDatabaseRestorePITR is the issue type for performing a Point-in-time Recovery.
	IssueDatabaseRestorePITR IssueType = "bb.issue.database.restore.pitr"
	// IssueDatabaseRollback is the issue type for a generated rollback issue.
	IssueDatabaseRollback IssueType = "bb.issue.database.rollback"
)

// IssueFieldID is the field ID for an issue.
// It has to be string type because the id for stage field contain multiple parts.
type IssueFieldID string

const (
	// IssueFieldName is the field ID for name.
	IssueFieldName IssueFieldID = "1"
	// IssueFieldStatus is the field ID for status.
	IssueFieldStatus IssueFieldID = "2"
	// IssueFieldAssignee is the field ID for assignee.
	IssueFieldAssignee IssueFieldID = "3"
	// IssueFieldDescription is the field ID for description.
	IssueFieldDescription IssueFieldID = "4"
	// IssueFieldProject is the field ID for project.
	IssueFieldProject IssueFieldID = "5"
	// IssueFieldSubscriberList is the field ID for subscriber list.
	IssueFieldSubscriberList IssueFieldID = "6"
	// IssueFieldSQL is the field ID for SQL.
	IssueFieldSQL IssueFieldID = "7"
)

// Issue is the API message for an issue.
type Issue struct {
	ID int `jsonapi:"primary,issue"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	ProjectID  int
	Project    *Project `jsonapi:"relation,project"`
	PipelineID int
	Pipeline   *Pipeline `jsonapi:"relation,pipeline"`

	// Domain specific fields
	Name                  string       `jsonapi:"attr,name"`
	Status                IssueStatus  `jsonapi:"attr,status"`
	Type                  IssueType    `jsonapi:"attr,type"`
	Description           string       `jsonapi:"attr,description"`
	AssigneeID            int          `jsonapi:"attr,assigneeId"`
	AssigneeNeedAttention bool         `jsonapi:"attr,assigneeNeedAttention"`
	Assignee              *Principal   `jsonapi:"relation,assignee"`
	SubscriberList        []*Principal `jsonapi:"relation,subscriberList"`
	Payload               string       `jsonapi:"attr,payload"`
}

// IssueResponse is the API message for an issue response.
type IssueResponse struct {
	Issues    []*Issue `jsonapi:"relation,issues"`
	NextToken string   `jsonapi:"attr,nextToken"`
}

// IssueCreate is the API message for creating an issue.
type IssueCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	ProjectID  int `jsonapi:"attr,projectId"`
	PipelineID int
	Pipeline   PipelineCreate `jsonapi:"attr,pipeline"`

	// Domain specific fields
	Name                  string    `jsonapi:"attr,name"`
	Type                  IssueType `jsonapi:"attr,type"`
	Description           string    `jsonapi:"attr,description"`
	AssigneeID            int       `jsonapi:"attr,assigneeId"`
	AssigneeNeedAttention bool      `jsonapi:"attr,assigneeNeedAttention"`
	SubscriberIDList      []int     `jsonapi:"attr,subscriberIdList"`
	Payload               string    `jsonapi:"attr,payload"`
	// CreateContext is used to create the issue pipeline and not persisted.
	// The context format depends on the issue type. For example, create database issue corresponds to CreateDatabaseContext.
	// This consolidates the pipeline generation to backend because both frontend and VCS pipeline could create issues and
	// we want the complexity resides in the backend.
	CreateContext string `jsonapi:"attr,createContext"`

	// ValidateOnly validates the request and previews the review, but does not actually post it.
	ValidateOnly bool `jsonapi:"attr,validateOnly"`
}

// CreateDatabaseContext is the issue create context for creating a database.
type CreateDatabaseContext struct {
	// InstanceID is the ID of an instance.
	InstanceID int `json:"instanceId"`
	// DatabaseName is the name of the database.
	DatabaseName string `json:"databaseName"`
	// TableName is the name of the table, if it is not empty, Bytebase should create a table after creating the database.
	// For example, in MongoDB, it only creates the database when we first store data in that database.
	TableName string `json:"tableName"`
	// CharacterSet is the character set of the database.
	CharacterSet string `json:"characterSet"`
	// Collation is the collation of the database.
	Collation string `json:"collation"`
	// Cluster is the cluster of the database. This is only applicable to ClickHouse for "ON CLUSTER <<cluster>>".
	Cluster string `json:"cluster"`
	// Owner is the owner of the database. This is only applicable to Postgres for "WITH OWNER <<owner>>".
	Owner string `json:"owner"`
	// BackupID is the ID of the backup.
	BackupID int `json:"backupId"`
	// Labels is a json-encoded string from a list of DatabaseLabel.
	// See definition in api.Database.
	Labels string `jsonapi:"attr,labels,omitempty"`
}

// MigrationDetail is the detail for database migration such as Migrate, Data.
type MigrationDetail struct {
	// MigrationType is the type of a migration.
	// MigrationType can be empty for gh-ost type of migration.
	MigrationType db.MigrationType `json:"migrationType"`
	// DatabaseID is the ID of a database.
	// This should be unset when a project is in tenant mode. The ProjectID is derived from IssueCreate.
	DatabaseID int `json:"databaseId"`
	// Statement is the statement to update database schema.
	Statement string `json:"statement"`
	// SheetID is the ID of a sheet. Statement and sheet ID is mutually exclusive.
	SheetID int `json:"sheetId"`
	// EarliestAllowedTs the earliest execution time of the change at system local Unix timestamp in seconds.
	EarliestAllowedTs int64 `jsonapi:"attr,earliestAllowedTs"`
	// SchemaVersion is parsed from VCS file name.
	// It is automatically generated in the UI workflow.
	SchemaVersion string `json:"schemaVersion"`
}

// MigrationContext is the issue create context for database migration such as Migrate, Data.
type MigrationContext struct {
	// DetailList is the details of schema update.
	// When a project is in tenant mode, there should be one item in the list.
	DetailList []*MigrationDetail `json:"detailList"`
	// VCSPushEvent is the event information for VCS push.
	VCSPushEvent *vcs.PushEvent `json:"vcsPushEvent"`
}

// MigrationFileYAMLDatabase contains the information of a database in a YAML
// format migration file.
type MigrationFileYAMLDatabase struct {
	Name string `yaml:"name"` // The name of the database
}

// MigrationFileYAML contains the information in a YAML format migration file.
type MigrationFileYAML struct {
	Databases []MigrationFileYAMLDatabase `yaml:"databases"` // The list of databases and how to identify them
	Statement string                      `yaml:"statement"` // The SQL statement to be executed to specified list of databases
}

// PITRContext is the issue create context for performing a PITR in a database.
type PITRContext struct {
	DatabaseID int `json:"databaseId"`

	// CreateDatabaseCtx will be not nil if user want to restore to a new database.
	// The new database should be under the same project as the original database.
	CreateDatabaseCtx *CreateDatabaseContext `json:"createDatabaseContext"`

	// BackupID and PointInTimeTs only allow one non-nil.

	// BackupID is not nil if the user just restore a full backup only.
	BackupID *int `json:"backupId"`

	// After the PITR operations, the database will be recovered to the state at this time.
	// Represented in UNIX timestamp in seconds.
	PointInTimeTs *int64 `json:"pointInTimeTs"`
}

// RollbackContext is the issue create context for rollback a issue.
type RollbackContext struct {
	// IssueID is the id of the issue to rollback.
	IssueID int `json:"issueId"`
	// TaskIDList is the list of task ids to rollback.
	TaskIDList []int `json:"taskIdList"`
}

// IssueFind is the API message for finding issues.
type IssueFind struct {
	ID *int

	// Related fields
	ProjectID *int

	// Domain specific fields
	PipelineID *int
	// Find issue where principalID is either creator, assignee or subscriber
	PrincipalID *int
	// To support pagination, we add into creator, assignee and subscriber.
	// Only principleID or one of the following three fields can be set.
	CreatorID             *int
	AssigneeID            *int
	AssigneeNeedAttention *bool
	SubscriberID          *int

	StatusList []IssueStatus
	// If specified, only find issues whose ID is smaller that SinceID.
	SinceID *int
	// If specified, then it will only fetch "Limit" most recently updated issues
	Limit *int
}

// IssuePatch is the API message for patching an issue.
type IssuePatch struct {
	ID int `jsonapi:"primary,issuePatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name *string `jsonapi:"attr,name"`
	// Status is only set manually via IssueStatusPatch
	Status                *IssueStatus
	Description           *string `jsonapi:"attr,description"`
	AssigneeID            *int    `jsonapi:"attr,assigneeId"`
	AssigneeNeedAttention *bool   `jsonapi:"attr,assigneeNeedAttention"`
	Payload               *string `jsonapi:"attr,payload"`
}

// IssueStatusPatch is the API message for patching status of an issue.
type IssueStatusPatch struct {
	ID int `jsonapi:"primary,issueStatusPatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Status  IssueStatus `jsonapi:"attr,status"`
	Comment string      `jsonapi:"attr,comment"`
}
