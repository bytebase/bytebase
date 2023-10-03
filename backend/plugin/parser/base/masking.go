package base

import (
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	DefaultMaskingLevel storepb.MaskingLevel = storepb.MaskingLevel_NONE
	MaxMaskingLevel     storepb.MaskingLevel = storepb.MaskingLevel_FULL
)

// SensitiveSchemaInfo is the schema info using to extract sensitive fields.
type SensitiveSchemaInfo struct {
	// IgnoreCaseSensitive is the flag to ignore case sensitive.
	// IMPORTANT: This flag is ONLY for database names, table names and view names in MySQL-like database.
	IgnoreCaseSensitive bool
	DatabaseList        []DatabaseSchema
}

// DatabaseSchema is the database schema using to extract sensitive fields.
type DatabaseSchema struct {
	Name       string
	SchemaList []SchemaSchema
}

// SchemaSchema is the schema of the schema using to extract sensitive fields.
type SchemaSchema struct {
	Name      string
	TableList []TableSchema
	ViewList  []ViewSchema
}

// ViewSchema is the view schema using to extract sensitive fields.
type ViewSchema struct {
	Name       string
	Definition string
}

// TableSchema is the table schema using to extract sensitive fields.
type TableSchema struct {
	Name       string
	ColumnList []ColumnInfo
}

// ColumnInfo is the column info using to extract sensitive fields.
type ColumnInfo struct {
	Name         string
	MaskingLevel storepb.MaskingLevel
}

// SensitiveField is the struct about SELECT fields.
type SensitiveField struct {
	Name         string
	MaskingLevel storepb.MaskingLevel
}

// FieldInfo is the masking field info.
type FieldInfo struct {
	Name         string
	Table        string
	Schema       string
	Database     string
	MaskingLevel storepb.MaskingLevel
}
