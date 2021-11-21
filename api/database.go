package api

import (
	"context"
	"database/sql"
	"encoding/json"
)

const (
	ALL_DATABASE_NAME          = "*"
	DEFAULT_CHARACTER_SET_NAME = "utf8mb4"
	// Use utf8mb4_general_ci instead of the new MySQL 8.0.1 default utf8mb4_0900_ai_ci
	// because the former is compatible with more other MySQL flavors (e.g. MariaDB)
	DEFAULT_COLLATION_NAME = "utf8mb4_general_ci"
)

type SyncStatus string

const (
	OK       SyncStatus = "OK"
	NotFound SyncStatus = "NOT_FOUND"
)

func (e SyncStatus) String() string {
	switch e {
	case OK:
		return "OK"
	case NotFound:
		return "NOT_FOUND"
	}
	return ""
}

type Database struct {
	ID int `jsonapi:"primary,database"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	ProjectID      int
	Project        *Project `jsonapi:"relation,project"`
	InstanceID     int
	Instance       *Instance     `jsonapi:"relation,instance"`
	DataSourceList []*DataSource `jsonapi:"relation,dataSource"`
	SourceBackupID int
	SourceBackup   *Backup `jsonapi:"relation,sourceBackup"`
	// Anomalies are stored in a separate table, but just return here for convenience
	AnomalyList []*Anomaly `jsonapi:"relation,anomaly"`

	// Domain specific fields
	Name                 string     `jsonapi:"attr,name"`
	CharacterSet         string     `jsonapi:"attr,characterSet"`
	Collation            string     `jsonapi:"attr,collation"`
	SyncStatus           SyncStatus `jsonapi:"attr,syncStatus"`
	LastSuccessfulSyncTs int64      `jsonapi:"attr,lastSuccessfulSyncTs"`
}

type DatabaseCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	ProjectID     int `jsonapi:"attr,projectID"`
	InstanceID    int `jsonapi:"attr,instanceID"`
	EnvironmentID int

	// Domain specific fields
	Name         string `jsonapi:"attr,name"`
	CharacterSet string `jsonapi:"attr,characterSet"`
	Collation    string `jsonapi:"attr,collation"`
	IssueID      int    `jsonapi:"attr,issueID"`
}

type DatabaseFind struct {
	ID *int

	// Related fields
	InstanceID *int
	ProjectID  *int

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
	UpdaterID int

	// Related fields
	ProjectID      *int `jsonapi:"attr,projectID"`
	SourceBackupID *int

	// Domain specific fields
	SyncStatus           *SyncStatus
	LastSuccessfulSyncTs *int64
}

type DatabaseService interface {
	CreateDatabase(ctx context.Context, create *DatabaseCreate) (*Database, error)
	// This is specifically used to create the * database when creating the instance.
	CreateDatabaseTx(ctx context.Context, tx *sql.Tx, create *DatabaseCreate) (*Database, error)
	FindDatabaseList(ctx context.Context, find *DatabaseFind) ([]*Database, error)
	FindDatabase(ctx context.Context, find *DatabaseFind) (*Database, error)
	PatchDatabase(ctx context.Context, patch *DatabasePatch) (*Database, error)
}
