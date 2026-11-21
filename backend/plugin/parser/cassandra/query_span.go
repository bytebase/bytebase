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
			Name:           result.Name,
			SourceColumns:  convertColumnSet(result.SourceColumns),
			IsPlainField:   result.IsPlainField,
			SelectAsterisk: result.SelectAsterisk,
		})
	}
	return converted
}
