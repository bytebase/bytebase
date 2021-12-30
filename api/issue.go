package api

import (
	"context"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
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
	// IssueDatabaseGrant is the issue type for granting databases.
	IssueDatabaseGrant IssueType = "bb.issue.database.grant"
	// IssueDatabaseSchemaUpdate is the issue type for updating database schemas.
	IssueDatabaseSchemaUpdate IssueType = "bb.issue.database.schema.update"
	// IssueDataSourceRequest is the issue type for requesting database sources.
	IssueDataSourceRequest IssueType = "bb.issue.data-source.request"
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
	// IssueFieldRollbackSQL is the field ID for rollback SQL.
	IssueFieldRollbackSQL IssueFieldID = "8"
)

// Issue is the API message for an issue.
type Issue struct {
	ID int `jsonapi:"primary,issue"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	ProjectID  int
	Project    *Project `jsonapi:"relation,project"`
	PipelineID int
	Pipeline   *Pipeline `jsonapi:"relation,pipeline"`

	// Domain specific fields
	Name             string      `jsonapi:"attr,name"`
	Status           IssueStatus `jsonapi:"attr,status"`
	Type             IssueType   `jsonapi:"attr,type"`
	Description      string      `jsonapi:"attr,description"`
	AssigneeID       int         `jsonapi:"attr,assigneeId"`
	Assignee         *Principal  `jsonapi:"attr,assignee"`
	SubscriberIDList []int       `jsonapi:"attr,subscriberIdList"`
	Payload          string      `jsonapi:"attr,payload"`
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
	Name             string    `jsonapi:"attr,name"`
	Type             IssueType `jsonapi:"attr,type"`
	Description      string    `jsonapi:"attr,description"`
	AssigneeID       int       `jsonapi:"attr,assigneeId"`
	SubscriberIDList []int     `jsonapi:"attr,subscriberIdList"`
	RollbackIssueID  *int      `jsonapi:"attr,rollbackIssueId"`
	Payload          string    `jsonapi:"attr,payload"`
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
	// CharacterSet is the character set of the database.
	CharacterSet string `json:"characterSet"`
	// Collation is the collation of the database.
	Collation string `json:"collation"`
	// BackupID is the ID of the backup.
	BackupID int `json:"backupId"`
	// BackupName is the name of the backup.
	BackupName string `json:"backupName"`
	// Labels is a json-encoded string from a list of DatabaseLabel.
	// See definition in api.Database.
	Labels string `jsonapi:"attr,labels,omitempty"`
}

// UpdateSchemaDetail is the detail of updating database schema.
type UpdateSchemaDetail struct {
	// DatabaseID is the ID of a database.
	DatabaseID int `json:"databaseId"`
	// DatabaseName is the name of databases, mutually exclusive to DatabaseID.
	// This should be set when a project is in tenant mode, and ProjectID is derived from IssueCreate.
	DatabaseName string `json:"databaseName"`
	// Statement is the statement to update database schema.
	Statement string `json:"statement"`
	// RollbackStatement is the rollback statement of the statement.
	RollbackStatement string `json:"rollbackStatement"`
	// EarliestAllowedTs the earliest execution time of the change at system local Unix timestamp in nanoseconds.
	EarliestAllowedTs int64 `jsonapi:"attr,earliestAllowedTs"`
}

// UpdateSchemaContext is the issue create context for updating database schema.
type UpdateSchemaContext struct {
	// MigrationType is the type of a migration.
	MigrationType db.MigrationType `json:"migrationType"`
	// UpdateSchemaDetail is the details of schema update.
	// When a project is in tenant mode, there should be one item in the list.
	UpdateSchemaDetailList []*UpdateSchemaDetail `json:"updateSchemaDetailList"`
	// VCSPushEvent is the event information for VCS push.
	VCSPushEvent *common.VCSPushEvent
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
	StatusList  *[]IssueStatus
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
	Status      *IssueStatus
	Description *string `jsonapi:"attr,description"`
	AssigneeID  *int    `jsonapi:"attr,assigneeId"`
	Payload     *string `jsonapi:"attr,payload"`
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

// IssueService is the services for issues.
type IssueService interface {
	CreateIssue(ctx context.Context, create *IssueCreate) (*Issue, error)
	FindIssueList(ctx context.Context, find *IssueFind) ([]*Issue, error)
	FindIssue(ctx context.Context, find *IssueFind) (*Issue, error)
	PatchIssue(ctx context.Context, patch *IssuePatch) (*Issue, error)
}
