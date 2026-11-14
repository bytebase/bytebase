package plsql

import (
	"context"
	"errors"

	"github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/oracle/analysis"
	pkgerrors "github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_ORACLE, GetQuerySpan)
}

func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	querySpan, err := resolveQuerySpan(ctx, gCtx, stmt.Text, database)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
}

func resolveQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, statement, database string) (*base.QuerySpan, error) {
	span, err := analysis.ResolveQuerySpan(statement, analysis.ResolveOptions{
		Instance:        gCtx.InstanceID,
		DefaultDatabase: database,
		Catalog:         newCatalog(ctx, gCtx),
	})
	if err != nil {
		if err == analysis.ErrMixUserSystemTables {
			return nil, base.MixUserSystemTablesError
		}
		if errors.Is(err, analysis.ErrMixUserSystemTables) {
			return nil, mixedTablesError{err}
		}
		return nil, err
	}
	if span == nil {
		return nil, nil
	}
	querySpan := &base.QuerySpan{
		Type:          convertQueryType(span.Type),
		SourceColumns: convertColumnSet(span.SourceColumns),
		Results:       convertResults(span.Results),
	}
	if span.NotFoundError != nil {
		querySpan.NotFoundError = convertNotFoundError(span.NotFoundError)
	}
	return querySpan, nil
}

// newCatalog reads synced metadata for omni. A nil metadata function stays
// nil, so omni fails where the extractor did.
func newCatalog(ctx context.Context, gCtx base.GetQuerySpanContext) analysis.Catalog {
	var catalog analysis.Catalog
	if gCtx.GetDatabaseMetadataFunc != nil {
		catalog.GetDatabase = func(instance, name string) (analysis.Database, error) {
			_, database, err := gCtx.GetDatabaseMetadataFunc(ctx, instance, name)
			if err != nil {
				return nil, toOmniError(err)
			}
			if database == nil {
				return nil, nil
			}
			return &catalogDatabase{database: database}, nil
		}
	}
	if gCtx.GetLinkedDatabaseMetadataFunc != nil {
		catalog.GetLinkedDatabase = func(instance, link, schema string) (string, string, analysis.Database, error) {
			linkedInstance, name, database, err := gCtx.GetLinkedDatabaseMetadataFunc(ctx, instance, link, schema)
			if err != nil {
				return "", "", nil, toOmniError(err)
			}
			if database == nil {
				return linkedInstance, name, nil, nil
			}
			return linkedInstance, name, &catalogDatabase{database: database}, nil
		}
	}
	return catalog
}

// catalogDatabase is a database's unnamed schema, where Oracle metadata keeps
// its objects.
type catalogDatabase struct {
	database *model.DatabaseMetadata
}

func (d *catalogDatabase) HasSchema() bool { return d.database.GetSchemaMetadata("") != nil }

func (d *catalogDatabase) GetTable(name string) *metadata.TableMetadata {
	table := d.database.GetSchemaMetadata("").GetTable(name)
	if table == nil {
		return nil
	}
	return table.GetProto()
}

func (d *catalogDatabase) GetView(name string) *metadata.ViewMetadata {
	return d.database.GetSchemaMetadata("").GetView(name)
}

func (d *catalogDatabase) GetMaterializedView(name string) *metadata.MaterializedViewMetadata {
	return d.database.GetSchemaMetadata("").GetMaterializedView(name)
}

func (d *catalogDatabase) GetExternalTable(name string) *metadata.ExternalTableMetadata {
	table := d.database.GetSchemaMetadata("").GetExternalTable(name)
	if table == nil {
		return nil
	}
	return table.GetProto()
}

// omniNotFoundError carries a lookup error that holds a ResourceNotFoundError
// into omni, which reports it on the span. It keeps the error's text and
// chain.
type omniNotFoundError struct {
	err      error
	notFound *analysis.NotFoundError
}

func (e omniNotFoundError) Error() string { return e.err.Error() }

func (e omniNotFoundError) Unwrap() error { return e.err }

func (e omniNotFoundError) As(target any) bool {
	if t, ok := target.(**analysis.NotFoundError); ok {
		*t = e.notFound
		return true
	}
	return false
}

func toOmniError(err error) error {
	var notFound *base.ResourceNotFoundError
	if !errors.As(err, &notFound) {
		return err
	}
	return omniNotFoundError{
		err: err,
		notFound: &analysis.NotFoundError{
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

// mixedTablesError is a resolution error that wraps omni's
// ErrMixUserSystemTables, as a view reading user and system tables does; it
// still matches base.MixUserSystemTablesError.
type mixedTablesError struct {
	error
}

func (mixedTablesError) Is(target error) bool { return target == base.MixUserSystemTablesError }

func (e mixedTablesError) Unwrap() error { return e.error }

func convertNotFoundError(err *analysis.NotFoundError) *base.ResourceNotFoundError {
	return &base.ResourceNotFoundError{
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

func convertQueryType(t analysis.QueryType) base.QueryType {
	switch t {
	case analysis.Select:
		return base.Select
	case analysis.Explain:
		return base.Explain
	case analysis.SelectInfoSchema:
		return base.SelectInfoSchema
	case analysis.DDL:
		return base.DDL
	case analysis.DML:
		return base.DML
	default:
		return base.QueryTypeUnknown
	}
}

func convertColumnSet(set analysis.ColumnSet) base.SourceColumnSet {
	if set == nil {
		return nil
	}
	result := make(base.SourceColumnSet, len(set))
	for column := range set {
		result[base.ColumnResource{
			Instance: column.Instance,
			Server:   column.Server,
			Database: column.Database,
			Schema:   column.Schema,
			Table:    column.Table,
			Column:   column.Column,
		}] = true
	}
	return result
}

func convertResults(results []analysis.QuerySpanResult) []base.QuerySpanResult {
	if results == nil {
		return nil
	}
	converted := make([]base.QuerySpanResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, base.QuerySpanResult{
			Name:          result.Name,
			SourceColumns: convertColumnSet(result.SourceColumns),
			IsPlainField:  result.IsPlainField,
			Invisible:     result.Invisible,
		})
	}
	return converted
}
