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

func convertResults(results []analysis.ResolvedQuerySpanResult) []base.QuerySpanResult {
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
