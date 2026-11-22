package base

import (
	"github.com/bytebase/omni/metadata"

	"github.com/bytebase/bytebase/backend/store/model"
)

// ConvertQueryType maps an engine-specific query-type enum value onto QueryType.
func ConvertQueryType[T comparable](t, sel, explain, infoSchema, ddl, dml T) QueryType {
	switch t {
	case sel:
		return Select
	case explain:
		return Explain
	case infoSchema:
		return SelectInfoSchema
	case ddl:
		return DDL
	case dml:
		return DML
	default:
		return QueryTypeUnknown
	}
}

// ConvertColumnSet converts a set of engine-specific source columns into SourceColumnSet.
func ConvertColumnSet[T comparable](set map[T]bool, convert func(T) ColumnResource) SourceColumnSet {
	if set == nil {
		return nil
	}
	out := make(SourceColumnSet, len(set))
	for col := range set {
		out[convert(col)] = true
	}
	return out
}

// ConvertQuerySpanResults maps engine-specific span result entries into []QuerySpanResult.
func ConvertQuerySpanResults[R any](results []R, convert func(R) QuerySpanResult) []QuerySpanResult {
	if results == nil {
		return nil
	}
	out := make([]QuerySpanResult, 0, len(results))
	for _, r := range results {
		out = append(out, convert(r))
	}
	return out
}

// MixedTablesError wraps an engine mixed-table error so errors.Is matches MixUserSystemTablesError.
type MixedTablesError struct {
	Cause error
}

// Error implements error.
func (e *MixedTablesError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return MixUserSystemTablesError.Error()
}

// Is allows errors.Is to match MixUserSystemTablesError.
func (*MixedTablesError) Is(target error) bool {
	return target == MixUserSystemTablesError
}

// Unwrap returns the underlying engine error.
func (e *MixedTablesError) Unwrap() error {
	return e.Cause
}

// UnnamedSchemaDatabase adapts a DatabaseMetadata's unnamed ("") schema to omni catalog interfaces.
type UnnamedSchemaDatabase struct {
	Database *model.DatabaseMetadata
}

// Name returns the database's protobuf name.
func (d *UnnamedSchemaDatabase) Name() string { return d.Database.GetProto().GetName() }

// HasSchema reports whether the unnamed schema exists.
func (d *UnnamedSchemaDatabase) HasSchema() bool {
	return d.Database.GetSchemaMetadata("") != nil
}

// ListTableNames lists table names in the unnamed schema.
func (d *UnnamedSchemaDatabase) ListTableNames() []string {
	return d.Database.GetSchemaMetadata("").ListTableNames()
}

// GetTable returns the table metadata proto in the unnamed schema, or nil if missing.
func (d *UnnamedSchemaDatabase) GetTable(name string) *metadata.TableMetadata {
	table := d.Database.GetSchemaMetadata("").GetTable(name)
	if table == nil {
		return nil
	}
	return table.GetProto()
}

// ListViewNames lists view names in the unnamed schema.
func (d *UnnamedSchemaDatabase) ListViewNames() []string {
	return d.Database.GetSchemaMetadata("").ListViewNames()
}

// GetView returns the view metadata proto in the unnamed schema, or nil if missing.
func (d *UnnamedSchemaDatabase) GetView(name string) *metadata.ViewMetadata {
	return d.Database.GetSchemaMetadata("").GetView(name)
}

// GetMaterializedView returns the materialized view metadata proto in the unnamed schema, or nil if missing.
func (d *UnnamedSchemaDatabase) GetMaterializedView(name string) *metadata.MaterializedViewMetadata {
	return d.Database.GetSchemaMetadata("").GetMaterializedView(name)
}

// GetExternalTable returns the external table metadata proto in the unnamed schema, or nil if missing.
func (d *UnnamedSchemaDatabase) GetExternalTable(name string) *metadata.ExternalTableMetadata {
	table := d.Database.GetSchemaMetadata("").GetExternalTable(name)
	if table == nil {
		return nil
	}
	return table.GetProto()
}
