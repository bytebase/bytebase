package base

import (
	"context"
	"errors"

	omnianalysis "github.com/bytebase/omni/analysis"
	"github.com/bytebase/omni/metadata"
	pkgerrors "github.com/pkg/errors"

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

// omniColumnResource converts an omni ColumnResource to a Bytebase ColumnResource.
func omniColumnResource(c omnianalysis.ColumnResource) ColumnResource {
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
		out[omniColumnResource(col)] = true
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

// omniNotFoundError wraps a lookup error carrying a ResourceNotFoundError so
// errors.As matches *omnianalysis.ResourceNotFoundError while preserving the
// original error's text and unwrap chain.
type omniNotFoundError struct {
	err      error
	notFound *omnianalysis.ResourceNotFoundError
}

func (e omniNotFoundError) Error() string { return e.err.Error() }

func (e omniNotFoundError) Unwrap() error { return e.err }

func (e omniNotFoundError) As(target any) bool {
	if t, ok := target.(**omnianalysis.ResourceNotFoundError); ok {
		*t = e.notFound
		return true
	}
	return false
}

// ToOmniNotFoundError adapts an error carrying a Bytebase ResourceNotFoundError
// so errors.As matches *omnianalysis.ResourceNotFoundError while keeping the
// original error's text and unwrap chain.
func ToOmniNotFoundError(err error) error {
	var notFound *ResourceNotFoundError
	if !errors.As(err, &notFound) {
		return err
	}
	return omniNotFoundError{
		err: err,
		notFound: &omnianalysis.ResourceNotFoundError{
			Err:          notFound.Err,
			Server:       notFound.Server,
			DatabaseLink: notFound.DatabaseLink,
			Database:     notFound.Database,
			Schema:       notFound.Schema,
			Table:        notFound.Table,
			Column:       notFound.Column,
			Function:     notFound.Function,
		},
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

// omniUnresolvedColumnsError converts an omni UnresolvedColumnsError to Bytebase's UnresolvedColumnsError.
func omniUnresolvedColumnsError(err *omnianalysis.UnresolvedColumnsError) *UnresolvedColumnsError {
	if err == nil {
		return nil
	}
	relations := make([]ColumnResource, 0, len(err.Relations))
	for _, relation := range err.Relations {
		relations = append(relations, omniColumnResource(relation))
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
		UnresolvedColumnsError: omniUnresolvedColumnsError(span.UnresolvedColumnsError),
	}
	if span.NotFoundError != nil {
		out.NotFoundError = OmniResourceNotFoundError(span.NotFoundError)
	}
	return out
}

// OmniResolvedSpanResult converts an omni ResolveQuerySpan (span, err) pair
// into a Bytebase QuerySpan, translating ErrMixUserSystemTables when present.
func OmniResolvedSpanResult(span *omnianalysis.ResolvedSpan, err error) (*QuerySpan, error) {
	if err != nil {
		if mixErr := ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		return nil, err
	}
	return OmniResolvedSpan(span), nil
}

// OmniQuerySpan builds a Bytebase QuerySpan from engine-local omni span fields
// whose NotFoundError is already an error interface.
func OmniQuerySpan(
	queryType omnianalysis.QueryType,
	sourceColumns, predicateColumns omnianalysis.ColumnSet,
	results []omnianalysis.QuerySpanResult,
	notFoundErr error,
) *QuerySpan {
	return &QuerySpan{
		Type:             OmniQueryType(queryType),
		SourceColumns:    OmniColumnSet(sourceColumns),
		PredicateColumns: OmniColumnSet(predicateColumns),
		Results:          OmniQuerySpanResults(results),
		NotFoundError:    OmniNotFoundError(notFoundErr),
	}
}

// OmniTableQuerySpan builds a table-level QuerySpan with initialized empty
// maps/slices for engines that report source tables without column lineage.
func OmniTableQuerySpan(queryType omnianalysis.QueryType, sourceColumns omnianalysis.ColumnSet) *QuerySpan {
	sources := OmniColumnSet(sourceColumns)
	if sources == nil {
		sources = SourceColumnSet{}
	}
	return &QuerySpan{
		Type:             OmniQueryType(queryType),
		SourceColumns:    sources,
		PredicateColumns: SourceColumnSet{},
		Results:          []QuerySpanResult{},
	}
}

// RequireDatabaseMetadata validates gCtx.GetDatabaseMetadataFunc and fetches
// synced metadata for database, returning an error if the callback is unset,
// fails, or reports an unsynced (nil) database.
func RequireDatabaseMetadata(ctx context.Context, gCtx GetQuerySpanContext, database string) (*model.DatabaseMetadata, error) {
	if gCtx.GetDatabaseMetadataFunc == nil {
		return nil, pkgerrors.New("GetDatabaseMetadataFunc is not set in GetQuerySpanContext")
	}
	_, meta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, database)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "failed to get database metadata for instance %q and database %q", gCtx.InstanceID, database)
	}
	if meta == nil {
		return nil, pkgerrors.Errorf("database metadata for database %q not found (database not synced)", database)
	}
	return meta, nil
}

// QuerySpanListDatabasesFunc adapts gCtx.ListDatabaseNamesFunc for omni
// query-span catalogs. Returns nil when ListDatabaseNamesFunc is nil.
func QuerySpanListDatabasesFunc(ctx context.Context, gCtx GetQuerySpanContext, translateNotFound bool) func() ([]string, error) {
	if gCtx.ListDatabaseNamesFunc == nil {
		return nil
	}
	return func() ([]string, error) {
		names, err := gCtx.ListDatabaseNamesFunc(ctx, gCtx.InstanceID)
		if err != nil && translateNotFound {
			return names, ToOmniNotFoundError(err)
		}
		return names, err
	}
}

// QuerySpanDatabaseGetter adapts gCtx.GetDatabaseMetadataFunc for omni
// query-span catalogs. Returns nil when GetDatabaseMetadataFunc is nil.
func QuerySpanDatabaseGetter[D any](
	ctx context.Context,
	gCtx GetQuerySpanContext,
	translateNotFound bool,
	wrap func(*model.DatabaseMetadata) D,
) func(string) (D, error) {
	if gCtx.GetDatabaseMetadataFunc == nil {
		return nil
	}
	return func(name string) (D, error) {
		var zero D
		_, database, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, name)
		if err != nil {
			if translateNotFound {
				return zero, ToOmniNotFoundError(err)
			}
			return zero, err
		}
		if database == nil {
			return zero, nil
		}
		return wrap(database), nil
	}
}

// QuerySpanProtoMetadataFunc adapts gCtx.GetDatabaseMetadataFunc to return raw
// omni DatabaseSchemaMetadata protos. Returns nil when unset.
func QuerySpanProtoMetadataFunc(ctx context.Context, gCtx GetQuerySpanContext, translateNotFound bool) func(string) (*metadata.DatabaseSchemaMetadata, error) {
	return QuerySpanDatabaseGetter(ctx, gCtx, translateNotFound, (*model.DatabaseMetadata).GetProto)
}

// UnnamedSchemaQuerySpanCatalog builds ListDatabaseNames and GetDatabase
// closures backed by gCtx for unnamed-schema engines, translating
// ResourceNotFoundError into omni's counterpart so omni reports it on the span.
func UnnamedSchemaQuerySpanCatalog[D any](
	ctx context.Context,
	gCtx GetQuerySpanContext,
	wrap func(*UnnamedSchemaDatabase) D,
) (list func() ([]string, error), get func(string) (D, error)) {
	list = QuerySpanListDatabasesFunc(ctx, gCtx, true)
	get = QuerySpanDatabaseGetter(ctx, gCtx, true, func(db *model.DatabaseMetadata) D {
		return wrap(&UnnamedSchemaDatabase{Database: db})
	})
	return list, get
}

// ConvertOmniMixTablesError translates omni's ErrMixUserSystemTables into Bytebase's
// MixUserSystemTablesError, preserving wrapping context via mixedTablesError.
// Returns nil if err does not match ErrMixUserSystemTables.
func ConvertOmniMixTablesError(err error) error {
	if err == omnianalysis.ErrMixUserSystemTables {
		return MixUserSystemTablesError
	}
	if errors.Is(err, omnianalysis.ErrMixUserSystemTables) {
		return &mixedTablesError{Cause: err}
	}
	return nil
}

// mixedTablesError wraps an engine mixed-table error so errors.Is matches MixUserSystemTablesError.
type mixedTablesError struct {
	Cause error
}

// Error implements error.
func (e *mixedTablesError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return MixUserSystemTablesError.Error()
}

// Is allows errors.Is to match MixUserSystemTablesError.
func (*mixedTablesError) Is(target error) bool {
	return target == MixUserSystemTablesError
}

// Unwrap returns the underlying engine error.
func (e *mixedTablesError) Unwrap() error {
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
