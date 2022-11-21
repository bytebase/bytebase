package api

import (
	"encoding/json"
)

const (
	// AllDatabaseName is the wild expression for all databases.
	AllDatabaseName = "*"
	// DefaultCharacterSetName is the default character set.
	DefaultCharacterSetName = "utf8mb4"
	// DefaultCollationName is the default collation name.
	// Use utf8mb4_general_ci instead of the new MySQL 8.0.1 default utf8mb4_0900_ai_ci
	// because the former is compatible with more other MySQL flavors (e.g. MariaDB).
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
	Name                 string `jsonapi:"attr,name"`
	CharacterSet         string `jsonapi:"attr,characterSet"`
	Collation            string `jsonapi:"attr,collation"`
	IssueID              int    `jsonapi:"attr,issueId"`
	LastSuccessfulSyncTs int64
	// Labels is a json-encoded string from a list of DatabaseLabel,
	// e.g. "[{"key":"bb.location","value":"earth"},{"key":"bb.tenant","value":"bytebase"}]".
	Labels        *string `jsonapi:"attr,labels"`
	SchemaVersion string
}

// DatabaseFind is the API message for finding databases.
type DatabaseFind struct {
	ID *int

	// Related fields
	ProjectID  *int
	InstanceID *int

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

// DatabaseEdit is the API message for updating a database in UI editor.
type DatabaseEdit struct {
	DatabaseID int `json:"databaseId"`

	CreateTableList []*CreateTableContext `json:"createTableList"`
	AlterTableList  []*AlterTableContext  `json:"alterTableList"`
	DropTableList   []*DropTableContext   `json:"dropTableList"`
}

// CreateTableContext is the edit database context to create a table.
type CreateTableContext struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Engine       string `json:"engine"`
	CharacterSet string `json:"characterSet"`
	Collation    string `json:"collation"`
	Comment      string `json:"comment"`

	AddColumnList []*AddColumnContext `json:"addColumnList"`
}

// AlterTableContext is the edit database context to alter a table.
type AlterTableContext struct {
	TableID int    `json:"tableId"`
	Name    string `json:"name"`

	// ColumnNameList should be the final order of columns in UI editor and is used to confirm column positions.
	ColumnNameList []string `json:"columnNameList"`

	AddColumnList   []*AddColumnContext   `json:"addColumnList"`
	AlterColumnList []*AlterColumnContext `json:"alterColumnList"`
	DropColumnList  []*DropColumnContext  `json:"dropColumnList"`
}

// DropTableContext is the edit database context to drop a table.
type DropTableContext struct {
	TableID int    `json:"tableId"`
	Name    string `json:"name"`
}

// AddColumnContext is the create/alter table context to add a column.
type AddColumnContext struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	CharacterSet string  `json:"characterSet"`
	Collation    string  `json:"collation"`
	Comment      string  `json:"comment"`
	Nullable     bool    `json:"nullable"`
	Default      *string `json:"default"`
}

// AlterColumnContext is the alter table context to alter a column.
type AlterColumnContext struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	CharacterSet string  `json:"characterSet"`
	Collation    string  `json:"collation"`
	Comment      string  `json:"comment"`
	Nullable     bool    `json:"nullable"`
	Default      *string `json:"default"`
}

// DropColumnContext is the alter table context to drop a column.
type DropColumnContext struct {
	Name string `json:"name"`
}
