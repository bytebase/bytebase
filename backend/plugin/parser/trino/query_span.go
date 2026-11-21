package trino

import (
	"context"
	stderrors "errors"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/trino/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_TRINO, GetQuerySpan)
}

// GetQuerySpan gets the query span for a Trino statement. It is the entry point
// registered with the base package.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, ignoreCaseSensitive bool) (*base.QuerySpan, error) {
	opts := analysis.ResolveOptions{
		DefaultDatabase:     database,
		DefaultSchema:       schema,
		IgnoreCaseSensitive: ignoreCaseSensitive,
	}
	if gCtx.ListDatabaseNamesFunc != nil {
		opts.ListDatabases = func() ([]string, error) {
			return gCtx.ListDatabaseNamesFunc(ctx, gCtx.InstanceID)
		}
	}
	if gCtx.GetDatabaseMetadataFunc != nil {
		opts.GetDatabaseMetadata = func(db string) (*metadatapb.DatabaseSchemaMetadata, error) {
			_, meta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, db)
			if err != nil {
				var notFound *base.ResourceNotFoundError
				if stderrors.As(err, &notFound) {
					return nil, &analysis.ResourceNotFoundError{
						Err:      notFound.Err,
						Database: notFound.Database,
						Schema:   notFound.Schema,
						Table:    notFound.Table,
						Column:   notFound.Column,
					}
				}
				return nil, err
			}
			if meta == nil {
				return nil, nil
			}
			return meta.GetProto(), nil
		}
	}

	resolved, err := analysis.ResolveQuerySpan(stmt.Text, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}

	results := base.ConvertQuerySpanResults(resolved.Results, func(r analysis.ResolvedQuerySpanResult) base.QuerySpanResult {
		return base.QuerySpanResult{
			Name:          r.Name,
			SourceColumns: toBaseColumnSet(r.SourceColumns),
			IsPlainField:  r.IsPlainField,
		}
	})

	span := &base.QuerySpan{
		Type:             mapQueryType(resolved.Type),
		SourceColumns:    toBaseColumnSet(resolved.SourceColumns),
		PredicateColumns: toBaseColumnSet(resolved.PredicateColumns),
		Results:          results,
	}
	if resolved.NotFoundError != nil {
		var nf *analysis.ResourceNotFoundError
		if stderrors.As(resolved.NotFoundError, &nf) {
			span.NotFoundError = &base.ResourceNotFoundError{
				Err:      nf.Err,
				Database: nf.Database,
				Schema:   nf.Schema,
				Table:    nf.Table,
				Column:   nf.Column,
			}
		} else {
			span.NotFoundError = &base.ResourceNotFoundError{
				Err: resolved.NotFoundError,
			}
		}
	}
	return span, nil
}

func mapQueryType(qt analysis.QueryType) base.QueryType {
	return base.ConvertQueryType(qt, analysis.Select, analysis.Explain, analysis.SelectInfoSchema, analysis.DDL, analysis.DML)
}

func toBaseColumnSet(cols analysis.ColumnSet) base.SourceColumnSet {
	return base.ConvertColumnSet(cols, func(col analysis.Column) base.ColumnResource {
		return base.ColumnResource{Database: col.Database, Schema: col.Schema, Table: col.Table, Column: col.Column}
	})
}
