package base

import (
	"github.com/bytebase/bytebase/backend/component/masker"
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
	MaskingAttributes MaskingAttributes
}

// SensitiveField is the struct about SELECT fields.
type SensitiveField struct {
	Name              string
	MaskingAttributes MaskingAttributes
}

// FieldInfo is the masking field info.
type FieldInfo struct {
	Name              string
	Table             string
	Schema            string
	Database          string
	MaskingAttributes MaskingAttributes
}

// MaskingAttributes contain the masking related attributes on the column, likes MaskingLevel.
type MaskingAttributes struct {
	Masker masker.Masker
}

// TransmittedBy transmits the masking attributes from other to self.
// If two masker is not equal, self will be set to default full masker.
func (m *MaskingAttributes) TransmittedBy(other MaskingAttributes) (changed bool) {
	changed = false
	defaultMasker := masker.NewDefaultFullMasker()
	if !m.Masker.Equal(other.Masker) && !m.Masker.Equal(defaultMasker) {
		m.Masker = defaultMasker
		changed = true
	}
	return changed
}

// TransmittedByInExpression transmits the masking attributes from other to self.
// If anyone of the masker is not none masker, both masker will be set to default full masker.
// It should be used in expression computing only.
func (m *MaskingAttributes) TransmittedByInExpression(other MaskingAttributes) (changed bool) {
	changed = false
	_, selfIsNoneMasker := m.Masker.(*masker.NoneMasker)
	_, otherIsNoneMasker := other.Masker.(*masker.NoneMasker)
	// Any none masker will be replaced with default full masker in expr computing.
	if (!selfIsNoneMasker) || (!otherIsNoneMasker) {
		m.Masker = masker.NewDefaultFullMasker()
		changed = true
	}
	return changed
}

// IsNeverChangeInTransmission returns true if the masking attributes would not never change in transmission, it can be used to do the quit early optimization.
func (m *MaskingAttributes) IsNeverChangeInTransmission() bool {
	_, ok := m.Masker.(*masker.FullMasker)
	return ok
}

// Clone clones the masking attributes.
func (m *MaskingAttributes) Clone() MaskingAttributes {
	return MaskingAttributes{
		Masker: m.Masker,
	}
}

// NewMaskingAttributes creates a new masking attributes.
func NewMaskingAttributes(masker masker.Masker) MaskingAttributes {
	return MaskingAttributes{
		Masker: masker,
	}
}

// NewDefaultMaskingAttributes creates a new masking attributes with default masking level.
func NewDefaultMaskingAttributes() MaskingAttributes {
	return NewMaskingAttributes(masker.NewNoneMasker())
}

// NewEmptyMaskingAttributes creates a new masking attributes with empty masking level.
func NewEmptyMaskingAttributes() MaskingAttributes {
	return NewMaskingAttributes(masker.NewNoneMasker())
}
