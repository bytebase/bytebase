package mysql

import (
	"context"
	"errors"

	"github.com/bytebase/omni/mysql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_MYSQL, GetQuerySpan)
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
			return nil, &base.MixedTablesError{Cause: err}
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
			return &base.UnnamedSchemaDatabase{Database: database}, nil
		}
	}
	return catalog
}

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
	return base.ConvertQueryType(t, analysis.Select, analysis.Explain, analysis.SelectInfoSchema, analysis.DDL, analysis.DML)
}

func convertColumnSet(set analysis.ColumnSet) base.SourceColumnSet {
	return base.ConvertColumnSet(set, func(c analysis.ColumnResource) base.ColumnResource {
		return base.ColumnResource{Database: c.Database, Schema: c.Schema, Table: c.Table, Column: c.Column}
	})
}

func convertResults(results []analysis.QuerySpanResult) []base.QuerySpanResult {
	return base.ConvertQuerySpanResults(results, func(r analysis.QuerySpanResult) base.QuerySpanResult {
		return base.QuerySpanResult{
			Name:          r.Name,
			SourceColumns: convertColumnSet(r.SourceColumns),
			IsPlainField:  r.IsPlainField,
		}
	})
}
