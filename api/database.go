package api

import (
	"encoding/json"
)

const (
	// AllDatabaseName is the wild expression for all databases.
	AllDatabaseName = "*"
	// DefaultCharactorSetName is the default character set.
	DefaultCharactorSetName = "utf8mb4"
	// DefaultCollationName is the default collation name.
	// Use utf8mb4_general_ci instead of the new MySQL 8.0.1 default utf8mb4_0900_ai_ci
	// because the former is compatible with more other MySQL flavors (e.g. MariaDB)
	DefaultCollationName = "utf8mb4_general_ci"
)

// SyncStatus is the database sync status.
type SyncStatus string

const (
	// OK is the OK sync status.
	OK SyncStatus = "OK"
	// NotFound is the NOT_FOUND sync status.
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

// Database is the API message for a database.
type Database struct {
	ID int `jsonapi:"primary,database"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	ProjectID      int           `jsonapi:"attr,projectId"`
	Project        *Project      `jsonapi:"relation,project"`
	InstanceID     int           `jsonapi:"attr,instanceId"`
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
	SchemaVersion        string     `jsonapi:"attr,schemaVersion"`
	SyncStatus           SyncStatus `jsonapi:"attr,syncStatus"`
	LastSuccessfulSyncTs int64      `jsonapi:"attr,lastSuccessfulSyncTs"`
	// Labels is a json-encoded string from a list of DatabaseLabel,
	// e.g. "[{"key":"bb.location","value":"earth"},{"key":"bb.tenant","value":"bytebase"}]".
	Labels string `jsonapi:"attr,labels,omitempty"`
}

// DatabaseCreate is the API message for creating a database.
type DatabaseCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	ProjectID     int `jsonapi:"attr,projectId"`
	InstanceID    int `jsonapi:"attr,instanceId"`
	EnvironmentID int

	// Domain specific fields
	Name         string `jsonapi:"attr,name"`
	CharacterSet string `jsonapi:"attr,characterSet"`
	Collation    string `jsonapi:"attr,collation"`
	IssueID      int    `jsonapi:"attr,issueId"`
	// Labels is a json-encoded string from a list of DatabaseLabel,
	// e.g. "[{"key":"bb.location","value":"earth"},{"key":"bb.tenant","value":"bytebase"}]".
	Labels        *string `jsonapi:"attr,labels"`
	SchemaVersion string
}

// DatabaseFind is the API message for finding databases.
type DatabaseFind struct {
	ID *int

	// Related fields
	ProjectID    *int
	NotProjectID *int
	InstanceID   *int

	// Domain specific fields
	Name               *string
	IncludeAllDatabase bool
	SyncStatus         *SyncStatus
}

func (find *DatabaseFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// DatabasePatch is the API message for patching a database.
type DatabasePatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Related fields
	ProjectID      *int `jsonapi:"attr,projectId"`
	SourceBackupID *int

	// Labels is a json-encoded string from a list of DatabaseLabel,
	// e.g. "[{"key":"bb.location","value":"earth"},{"key":"bb.tenant","value":"bytebase"}]".
	Labels *string `jsonapi:"attr,labels"`

	// Domain specific fields
	SchemaVersion        *string
	SyncStatus           *SyncStatus
	LastSuccessfulSyncTs *int64
}
