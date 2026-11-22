package redshift

import (
	"context"
	"errors"

	"github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/redshift/analysis"
	pkgerrors "github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_REDSHIFT, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, _ bool) (*base.QuerySpan, error) {
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
	searchPath := meta.GetSearchPath()
	if schema != "" {
		searchPath = []string{schema}
	}

	span, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase: database,
		SearchPath:      searchPath,
		GetDatabaseMetadata: func(db string) (*metadata.DatabaseSchemaMetadata, error) {
			_, dbMeta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, db)
			if err != nil {
				return nil, err
			}
			if dbMeta == nil {
				return nil, nil
			}
			return dbMeta.GetProto(), nil
		},
	})
	if err != nil {
		if errors.Is(err, analysis.ErrMixUserSystemTables) {
			return nil, base.MixUserSystemTablesError
		}
		err = convertError(err, stmt)
		return nil, pkgerrors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}

	return &base.QuerySpan{
		Type:                      convertQueryType(span.Type),
		SourceColumns:             convertColumnSet(span.SourceColumns),
		PredicateColumns:          convertColumnSet(span.PredicateColumns),
		Results:                   convertResults(span.Results),
		NotFoundError:             convertNotFoundError(span.NotFoundError),
		FunctionNotSupportedError: convertFunctionNotSupportedError(span.FunctionNotSupportedError),
	}, nil
}

func convertQueryType(t analysis.QueryType) base.QueryType {
	return base.ConvertQueryType(t, analysis.Select, analysis.Explain, analysis.SelectInfoSchema, analysis.DDL, analysis.DML)
}

func convertColumnSet(set analysis.ColumnSet) base.SourceColumnSet {
	return base.ConvertColumnSet(set, func(c analysis.ColumnResource) base.ColumnResource {
		return base.ColumnResource{Database: c.Database, Schema: c.Schema, Table: c.Table, Column: c.Column}
	})
}

func convertResults(results []analysis.ResolvedQuerySpanResult) []base.QuerySpanResult {
	return base.ConvertQuerySpanResults(results, func(r analysis.ResolvedQuerySpanResult) base.QuerySpanResult {
		return base.QuerySpanResult{
			Name:          r.Name,
			SourceColumns: convertColumnSet(r.SourceColumns),
			IsPlainField:  r.IsPlainField,
		}
	})
}

func convertNotFoundError(err error) error {
	if err == nil {
		return nil
	}
	var notFound *analysis.ResourceNotFoundError
	if errors.As(err, &notFound) {
		return &base.ResourceNotFoundError{
			Err:      notFound.Err,
			Database: notFound.Database,
			Schema:   notFound.Schema,
			Table:    notFound.Table,
			Column:   notFound.Column,
		}
	}
	return err
}

func convertFunctionNotSupportedError(err error) error {
	if err == nil {
		return nil
	}
	var unsupported *analysis.FunctionNotSupportedError
	if errors.As(err, &unsupported) {
		return &base.FunctionNotSupportedError{
			Err:        unsupported.Err,
			Function:   unsupported.Function,
			Definition: unsupported.Definition,
		}
	}
	return err
}
