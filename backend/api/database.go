package api

import "context"

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

	// Related fields
	Project        *Project `jsonapi:"relation,project"`
	ProjectId      int
	Instance       *Instance `jsonapi:"relation,instance"`
	InstanceId     int
	DataSourceList []*DataSource `jsonapi:"relation,dataSource"`

	// Standard fields
	WorkspaceId int
	CreatorId   int   `jsonapi:"attr,creatorId"`
	CreatedTs   int64 `jsonapi:"attr,createdTs"`
	UpdaterId   int   `jsonapi:"attr,updaterId"`
	UpdatedTs   int64 `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name                 string     `jsonapi:"attr,name"`
	SyncStatus           SyncStatus `jsonapi:"attr,syncStatus"`
	LastSuccessfulSyncTs int64      `jsonapi:"attr,lastSuccessfulSyncTs"`
	Fingerprint          string     `jsonapi:"attr,fingerprint"`
}

type DatabaseCreate struct {
	// Related fields
	ProjectId  int
	InstanceId int

	// Standard fields
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Domain specific fields
	Name    string `jsonapi:"attr,name"`
	IssueId string `jsonapi:"attr,issueId"`
}

type DatabaseFind struct {
	// Standard fields
	ID                 *int
	WorkspaceId        *int
	InstanceId         *int
	IncludeAllDatabase bool
}

type DatabasePatch struct {
	// Related fields
	ProjectId *int `jsonapi:"attr,project"`

	// Standard fields
	ID          int `jsonapi:"primary,database-patch"`
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int
}

type DatabaseService interface {
	CreateDatabase(ctx context.Context, create *DatabaseCreate) (*Database, error)
	FindDatabaseList(ctx context.Context, find *DatabaseFind) ([]*Database, error)
	FindDatabase(ctx context.Context, find *DatabaseFind) (*Database, error)
	PatchDatabaseByID(ctx context.Context, patch *DatabasePatch) (*Database, error)
}
