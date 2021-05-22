package api

import (
	"context"
	"encoding/json"
)

const ALL_DATABASE_NAME = "*"

type SyncStatus string

const (
	OK       SyncStatus = "OK"
	Drifted  SyncStatus = "DRIFTED"
	NotFound SyncStatus = "NOT_FOUND"
)

func (e SyncStatus) String() string {
	switch e {
	case OK:
		return "OK"
	case Drifted:
		return "DRIFTED"
	case NotFound:
		return "NOT_FOUND"
	}
	return ""
}

type Database struct {
	ID int `jsonapi:"primary,database"`

	// Standard fields
	CreatorId   int
	Creator     *Principal `jsonapi:"attr,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"attr,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Related fields
	ProjectId      int
	Project        *Project `jsonapi:"relation,project"`
	InstanceId     int
	Instance       *Instance     `jsonapi:"relation,instance"`
	DataSourceList []*DataSource `jsonapi:"relation,dataSource"`

	// Domain specific fields
	Name                 string     `jsonapi:"attr,name"`
	SyncStatus           SyncStatus `jsonapi:"attr,syncStatus"`
	LastSuccessfulSyncTs int64      `jsonapi:"attr,lastSuccessfulSyncTs"`
	Fingerprint          string     `jsonapi:"attr,fingerprint"`
}

type DatabaseCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Related fields
	ProjectId  int
	InstanceId int

	// Domain specific fields
	Name    string `jsonapi:"attr,name"`
	IssueId int    `jsonapi:"attr,issueId"`
}

type DatabaseFind struct {
	ID *int

	// Standard fields
	WorkspaceId *int

	// Related fields
	InstanceId *int
	ProjectId  *int

	// Domain specific fields
	Name               *string
	IncludeAllDatabase bool
}

func (find *DatabaseFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type DatabasePatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

	// Related fields
	ProjectId *int `jsonapi:"attr,projectId"`

	// Domain specific fields
	SyncStatus           *SyncStatus
	LastSuccessfulSyncTs *int64
}

type DatabaseService interface {
	CreateDatabase(ctx context.Context, create *DatabaseCreate) (*Database, error)
	FindDatabaseList(ctx context.Context, find *DatabaseFind) ([]*Database, error)
	FindDatabase(ctx context.Context, find *DatabaseFind) (*Database, error)
	PatchDatabase(ctx context.Context, patch *DatabasePatch) (*Database, error)
}
