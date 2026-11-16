package oceanbase

import (
	"context"
	"errors"

	"github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/oceanbase/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_OCEANBASE, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	stmt base.Statement,
	database, _ string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	querySpan, err := resolveQuerySpan(ctx, gCtx, stmt.Text, database, ignoreCaseSensitive)
	if err != nil {
		return nil, convertError(err, stmt)
	}
	return querySpan, nil
}

func resolveQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, statement, database string, ignoreCaseSensitive bool) (*base.QuerySpan, error) {
	span, err := analysis.ResolveQuerySpan(statement, analysis.ResolveOptions{
		DefaultDatabase:     database,
		IgnoreCaseSensitive: ignoreCaseSensitive,
		Catalog:             newCatalog(ctx, gCtx),
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
	querySpan := &base.QuerySpan{
		Type:             convertQueryType(span.Type),
		ExplainAnalyze:   span.ExplainAnalyze,
		SourceColumns:    convertColumnSet(span.SourceColumns),
		Results:          convertResults(span.Results),
		PredicateColumns: convertColumnSet(span.PredicateColumns),
	}
	if span.NotFoundError != nil {
		querySpan.NotFoundError = convertNotFoundError(span.NotFoundError)
	}
	return querySpan, nil
}

// newCatalog reads the instance's synced metadata for omni. A lookup error
// carrying a ResourceNotFoundError becomes omni's NotFoundError, which omni
// reports on the span rather than as an error.
func newCatalog(ctx context.Context, gCtx base.GetQuerySpanContext) analysis.Catalog {
	var catalog analysis.Catalog
	if gCtx.ListDatabaseNamesFunc != nil {
		catalog.ListDatabaseNames = func() ([]string, error) {
			names, err := gCtx.ListDatabaseNamesFunc(ctx, gCtx.InstanceID)
			return names, toOmniNotFound(err)
		}
	}
	if gCtx.GetDatabaseMetadataFunc != nil {
		catalog.GetDatabase = func(name string) (analysis.Database, error) {
			_, database, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, name)
			if err != nil {
				return nil, toOmniNotFound(err)
			}
			if database == nil {
				return nil, nil
			}
			return &catalogDatabase{database: database}, nil
		}
	}
	return catalog
}

// catalogDatabase is a database's unnamed schema, where OceanBase metadata keeps
// tables and views.
type catalogDatabase struct {
	database *model.DatabaseMetadata
}

func (d *catalogDatabase) Name() string { return d.database.GetProto().GetName() }

func (d *catalogDatabase) HasSchema() bool { return d.database.GetSchemaMetadata("") != nil }

func (d *catalogDatabase) ListTableNames() []string {
	return d.database.GetSchemaMetadata("").ListTableNames()
}

func (d *catalogDatabase) GetTable(name string) *metadata.TableMetadata {
	table := d.database.GetSchemaMetadata("").GetTable(name)
	if table == nil {
		return nil
	}
	return table.GetProto()
}

func (d *catalogDatabase) ListViewNames() []string {
	return d.database.GetSchemaMetadata("").ListViewNames()
}

func (d *catalogDatabase) GetView(name string) *metadata.ViewMetadata {
	return d.database.GetSchemaMetadata("").GetView(name)
}

// mixedTablesError is a resolution error that wraps omni's
// ErrMixUserSystemTables, as a view reading user and system tables does; it
// still matches base.MixUserSystemTablesError.
type mixedTablesError struct {
	error
}

func (mixedTablesError) Is(target error) bool { return target == base.MixUserSystemTablesError }

func (e mixedTablesError) Unwrap() error { return e.error }

func toOmniNotFound(err error) error {
	var notFound *base.ResourceNotFoundError
	if !errors.As(err, &notFound) {
		return err
	}
	return &analysis.NotFoundError{
		Err:      notFound.Err,
		Database: notFound.Database,
		Schema:   notFound.Schema,
		Table:    notFound.Table,
		Column:   notFound.Column,
	}
}

func convertNotFoundError(err *analysis.NotFoundError) *base.ResourceNotFoundError {
	return &base.ResourceNotFoundError{
		Err:      err.Err,
		Database: err.Database,
		Schema:   err.Schema,
		Table:    err.Table,
		Column:   err.Column,
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
		})
	}
	return converted
}
