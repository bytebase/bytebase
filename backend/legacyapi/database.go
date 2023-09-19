package api

import (
	"encoding/json"
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

	// Related fields
	ProjectID      int           `jsonapi:"attr,projectId"`
	Project        *Project      `jsonapi:"relation,project"`
	InstanceID     int           `jsonapi:"attr,instanceId"`
	Instance       *Instance     `jsonapi:"relation,instance"`
	DataSourceList []*DataSource `jsonapi:"relation,dataSource"`

	// Domain specific fields
	Name                 string     `jsonapi:"attr,name"`
	SchemaVersion        string     `jsonapi:"attr,schemaVersion"`
	SyncStatus           SyncStatus `jsonapi:"attr,syncStatus"`
	LastSuccessfulSyncTs int64      `jsonapi:"attr,lastSuccessfulSyncTs"`
	// Labels is a json-encoded string from a list of DatabaseLabel,
	// e.g. "[{"key":"bb.location","value":"earth"},{"key":"bb.tenant","value":"bytebase"}]".
	Labels string `jsonapi:"attr,labels,omitempty"`
}

// DatabaseFind is the API message for finding databases.
type DatabaseFind struct {
	ID *int

	// Related fields
	ProjectID  *int
	InstanceID *int

	// Domain specific fields
	Name       *string
	SyncStatus *SyncStatus
}

func (find *DatabaseFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// DatabaseEdit is the API message for updating a database in UI editor.
type DatabaseEdit struct {
	DatabaseID int `json:"databaseId"`

	CreateSchemaList []*CreateSchemaContext `json:"createSchemaList"`
	RenameSchemaList []*RenameSchemaContext `json:"renameSchemaList"`
	DropSchemaList   []*DropSchemaContext   `json:"dropSchemaList"`
	CreateTableList  []*CreateTableContext  `json:"createTableList"`
	AlterTableList   []*AlterTableContext   `json:"alterTableList"`
	RenameTableList  []*RenameTableContext  `json:"renameTableList"`
	DropTableList    []*DropTableContext    `json:"dropTableList"`
}

// CreateSchemaContext is the edit database context to create a schema.
type CreateSchemaContext struct {
	Schema string `json:"schema"`
}

// RenameSchemaContext is the edit database context to rename a schema.
type RenameSchemaContext struct {
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}

// DropSchemaContext is the edit database context to drop a schema.
type DropSchemaContext struct {
	Schema string `json:"schema"`
}

// CreateTableContext is the edit database context to create a table.
type CreateTableContext struct {
	Schema       string `json:"schema"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Engine       string `json:"engine"`
	CharacterSet string `json:"characterSet"`
	Collation    string `json:"collation"`
	Comment      string `json:"comment"`

	AddColumnList     []*AddColumnContext     `json:"addColumnList"`
	PrimaryKeyList    []string                `json:"primaryKeyList"`
	AddForeignKeyList []*AddForeignKeyContext `json:"addForeignKeyList"`
}

// AlterTableContext is the edit database context to alter a table.
type AlterTableContext struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`

	// ColumnNameList should be the final order of columns in UI editor and is used to confirm column positions.
	ColumnNameList []string `json:"columnNameList"`

	AddColumnList    []*AddColumnContext    `json:"addColumnList"`
	AlterColumnList  []*AlterColumnContext  `json:"alterColumnList"`
	ChangeColumnList []*ChangeColumnContext `json:"changeColumnList"`
	DropColumnList   []*DropColumnContext   `json:"dropColumnList"`
	// TODO(steven): diff schemas in backend, so we don't need this flag.
	DropPrimaryKey     bool                    `json:"dropPrimaryKey"`
	DropPrimaryKeyList []string                `json:"dropPrimaryKeyList"`
	PrimaryKeyList     *[]string               `json:"primaryKeyList"`
	DropForeignKeyList []string                `json:"dropForeignKeyList"`
	AddForeignKeyList  []*AddForeignKeyContext `json:"addForeignKeyList"`
}

// RenameTableContext is the edit database context to rename a table.
type RenameTableContext struct {
	Schema  string `json:"schema"`
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}

// DropTableContext is the edit database context to drop a table.
type DropTableContext struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
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
	OldName        string  `json:"oldName"`
	NewName        string  `json:"newName"`
	Type           *string `json:"type"`
	Comment        *string `json:"comment"`
	Nullable       *bool   `json:"nullable"`
	DefaultChanged bool    `json:"defaultChanged"`
	Default        *string `json:"default"`
}

// ChangeColumnContext is the alter table context to change a column, mainly used in MySQL.
type ChangeColumnContext struct {
	OldName      string  `json:"oldName"`
	NewName      string  `json:"newName"`
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

// AddForeignKeyContext is the add foreign key context.
type AddForeignKeyContext struct {
	ColumnList           []string `json:"columnList"`
	ReferencedSchema     string   `json:"referencedSchema"`
	ReferencedTable      string   `json:"referencedTable"`
	ReferencedColumnList []string `json:"referencedColumnList"`
}

// ValidateResultType is the type of a validate result.
type ValidateResultType string

const (
	// ValidateErrorResult is the validate result type for ERROR validation.
	ValidateErrorResult ValidateResultType = "ERROR"
)

// ValidateResult is a validation result type, including validation type and message.
type ValidateResult struct {
	Type    ValidateResultType `json:"type"`
	Message string             `json:"message"`
}

func (result *ValidateResult) String() string {
	str, err := json.Marshal(*result)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// DatabaseEditResult is the response api message for editing database.
type DatabaseEditResult struct {
	Statement          string            `jsonapi:"attr,statement"`
	ValidateResultList []*ValidateResult `jsonapi:"attr,validateResultList"`
}
