package base

import (
	"errors"

	omnianalysis "github.com/bytebase/omni/analysis"
	"github.com/bytebase/omni/metadata"

	"github.com/bytebase/bytebase/backend/store/model"
)

// OmniQueryType maps an omni shared analysis.QueryType onto QueryType.
func OmniQueryType(t omnianalysis.QueryType) QueryType {
	switch t {
	case omnianalysis.Select:
		return Select
	case omnianalysis.Explain:
		return Explain
	case omnianalysis.SelectInfoSchema:
		return SelectInfoSchema
	case omnianalysis.DDL:
		return DDL
	case omnianalysis.DML:
		return DML
	default:
		return QueryTypeUnknown
	}
}

// OmniColumnResource converts an omni ColumnResource to a Bytebase ColumnResource.
func OmniColumnResource(c omnianalysis.ColumnResource) ColumnResource {
	return ColumnResource{
		Instance: c.Instance,
		Server:   c.Server,
		Database: c.Database,
		Schema:   c.Schema,
		Table:    c.Table,
		Column:   c.Column,
	}
}

// OmniColumnSet converts an omni ColumnSet into SourceColumnSet.
func OmniColumnSet(set omnianalysis.ColumnSet) SourceColumnSet {
	if set == nil {
		return nil
	}
	out := make(SourceColumnSet, len(set))
	for col := range set {
		out[OmniColumnResource(col)] = true
	}
	return out
}

// OmniQuerySpanResults converts omni QuerySpanResult entries into []QuerySpanResult.
func OmniQuerySpanResults(results []omnianalysis.QuerySpanResult) []QuerySpanResult {
	if results == nil {
		return nil
	}
	out := make([]QuerySpanResult, 0, len(results))
	for _, r := range results {
		out = append(out, QuerySpanResult{
			Name:           r.Name,
			SourceColumns:  OmniColumnSet(r.SourceColumns),
			IsPlainField:   r.IsPlainField,
			SelectAsterisk: r.SelectAsterisk,
			Invisible:      r.Invisible,
		})
	}
	return out
}

// OmniResourceNotFoundError converts an omni ResourceNotFoundError to Bytebase's ResourceNotFoundError.
func OmniResourceNotFoundError(err *omnianalysis.ResourceNotFoundError) *ResourceNotFoundError {
	if err == nil {
		return nil
	}
	return &ResourceNotFoundError{
		Err:          err.Err,
		Server:       err.Server,
		DatabaseLink: err.DatabaseLink,
		Database:     err.Database,
		Schema:       err.Schema,
		Table:        err.Table,
		Column:       err.Column,
		Function:     err.Function,
	}
}

// OmniNotFoundError translates an error carrying an omni ResourceNotFoundError into Bytebase's ResourceNotFoundError.
func OmniNotFoundError(err error) error {
	if err == nil {
		return nil
	}
	var notFound *omnianalysis.ResourceNotFoundError
	if errors.As(err, &notFound) {
		return OmniResourceNotFoundError(notFound)
	}
	return err
}

// ToOmniNotFoundError converts a Bytebase ResourceNotFoundError into an omni ResourceNotFoundError.
func ToOmniNotFoundError(err error) error {
	var notFound *ResourceNotFoundError
	if !errors.As(err, &notFound) {
		return err
	}
	return &omnianalysis.ResourceNotFoundError{
		Err:          notFound.Err,
		Server:       notFound.Server,
		DatabaseLink: notFound.DatabaseLink,
		Database:     notFound.Database,
		Schema:       notFound.Schema,
		Table:        notFound.Table,
		Column:       notFound.Column,
		Function:     notFound.Function,
	}
}

// OmniFunctionNotSupportedError translates an error carrying an omni FunctionNotSupportedError into Bytebase's type.
func OmniFunctionNotSupportedError(err error) error {
	if err == nil {
		return nil
	}
	var unsupported *omnianalysis.FunctionNotSupportedError
	if errors.As(err, &unsupported) {
		return &FunctionNotSupportedError{
			Err:        unsupported.Err,
			Function:   unsupported.Function,
			Definition: unsupported.Definition,
		}
	}
	return err
}

// OmniTypeNotSupportedError converts an omni TypeNotSupportedError to Bytebase's TypeNotSupportedError.
func OmniTypeNotSupportedError(err *omnianalysis.TypeNotSupportedError) *TypeNotSupportedError {
	if err == nil {
		return nil
	}
	return &TypeNotSupportedError{
		Err:   err.Err,
		Type:  err.Type,
		Name:  err.Name,
		Extra: err.Extra,
	}
}

// OmniUnresolvedColumnsError converts an omni UnresolvedColumnsError to Bytebase's UnresolvedColumnsError.
func OmniUnresolvedColumnsError(err *omnianalysis.UnresolvedColumnsError) *UnresolvedColumnsError {
	if err == nil {
		return nil
	}
	relations := make([]ColumnResource, 0, len(err.Relations))
	for _, relation := range err.Relations {
		relations = append(relations, OmniColumnResource(relation))
	}
	return &UnresolvedColumnsError{Relations: relations}
}

// OmniResolvedSpan converts an omni ResolvedSpan into a Bytebase QuerySpan.
func OmniResolvedSpan(span *omnianalysis.ResolvedSpan) *QuerySpan {
	if span == nil {
		return nil
	}
	out := &QuerySpan{
		Type:                   OmniQueryType(span.Type),
		ExplainAnalyze:         span.ExplainAnalyze,
		SourceColumns:          OmniColumnSet(span.SourceColumns),
		Results:                OmniQuerySpanResults(span.Results),
		PredicateColumns:       OmniColumnSet(span.PredicateColumns),
		UnresolvedColumnsError: OmniUnresolvedColumnsError(span.UnresolvedColumnsError),
	}
	if span.NotFoundError != nil {
		out.NotFoundError = OmniResourceNotFoundError(span.NotFoundError)
	}
	return out
}

// ConvertOmniMixTablesError translates omni's ErrMixUserSystemTables into Bytebase's
// MixUserSystemTablesError, preserving wrapping context via MixedTablesError.
// Returns nil if err does not match ErrMixUserSystemTables.
func ConvertOmniMixTablesError(err error) error {
	if err == omnianalysis.ErrMixUserSystemTables {
		return MixUserSystemTablesError
	}
	if errors.Is(err, omnianalysis.ErrMixUserSystemTables) {
		return &MixedTablesError{Cause: err}
	}
	return nil
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
