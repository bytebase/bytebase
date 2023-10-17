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
	Name              string
	MaskingAttributes *MaskingAttributes
}

// SensitiveField is the struct about SELECT fields.
type SensitiveField struct {
	Name              string
	MaskingAttributes *MaskingAttributes
}

// FieldInfo is the masking field info.
type FieldInfo struct {
	Name             string
	Table            string
	Schema           string
	Database         string
	MaskingAttrbutes *MaskingAttributes
}

// MaskingAttributes contain the masking related attributes on the column, likes MaskingLevel.
type MaskingAttributes struct {
	MaskingLevel storepb.MaskingLevel
}

// Transmit transmits the masking attributes to other.
func (m *MaskingAttributes) Transmit(other *MaskingAttributes) {
	other.MaskingLevel = m.MaskingLevel
}

// Clone clones the masking attributes.
func (m *MaskingAttributes) Clone() *MaskingAttributes {
	return &MaskingAttributes{
		MaskingLevel: m.MaskingLevel,
	}
}

// NewMaskingAttributes creates a new masking attributes.
func NewMaskingAttributes(lvl storepb.MaskingLevel) *MaskingAttributes {
	return &MaskingAttributes{
		MaskingLevel: lvl,
	}
}

func NewDefaultMaskingAttributes() *MaskingAttributes {
	return NewMaskingAttributes(DefaultMaskingLevel)
}
