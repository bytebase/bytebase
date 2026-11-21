package cassandra

import (
	"context"

	"github.com/bytebase/omni/cassandra/analysis"
	"github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_CASSANDRA, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given CQL statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	opts := analysis.ResolveOptions{
		DefaultKeyspace: database,
	}
	if gCtx.GetDatabaseMetadataFunc != nil {
		opts.GetDatabaseMetadata = func(keyspace string) (*metadata.DatabaseSchemaMetadata, error) {
			_, meta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, keyspace)
			if err != nil {
				return nil, err
			}
			if meta == nil {
				return nil, nil
			}
			return meta.GetProto(), nil
		}
	}

	span, err := analysis.ResolveQuerySpan(stmt.Text, opts)
	if err != nil {
		return nil, convertError(err, base.Statement{Text: stmt.Text})
	}
	return &base.QuerySpan{
		Type:             convertQueryType(span.Type),
		SourceColumns:    convertColumnSet(span.SourceColumns),
		PredicateColumns: convertColumnSet(span.PredicateColumns),
		Results:          convertResults(span.Results),
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
			Name:           r.Name,
			SourceColumns:  convertColumnSet(r.SourceColumns),
			IsPlainField:   r.IsPlainField,
			SelectAsterisk: r.SelectAsterisk,
		}
	})
}
