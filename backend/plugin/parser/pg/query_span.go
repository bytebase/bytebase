package pg

import (
	"context"

	"github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/analysis"
	omniparser "github.com/bytebase/omni/pg/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_POSTGRES, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_COCKROACHDB, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, _ bool) (*base.QuerySpan, error) {
	if gCtx.GetDatabaseMetadataFunc == nil {
		return nil, errors.New("GetDatabaseMetadataFunc is not set in GetQuerySpanContext")
	}
	_, meta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for instance %q and database %q", gCtx.InstanceID, database)
	}
	if meta == nil {
		return nil, errors.Errorf("database metadata for database %q not found (database not synced)", database)
	}
	searchPath := meta.GetSearchPath()
	if schema != "" {
		searchPath = analysis.SearchPathForSelectedSchema(schema, searchPath)
	}

	querySpan, err := resolveQuerySpan(ctx, gCtx, stmt.Text, database, searchPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
}

func resolveQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, statement, database string, searchPath []string) (*base.QuerySpan, error) {
	span, err := analysis.ResolveQuerySpan(ctx, statement, analysis.ResolveOptions{
		DefaultDatabase: database,
		SearchPath:      searchPath,
		Catalog:         newCatalog(ctx, gCtx),
	})
	if err != nil {
		if syntaxErr, ok := err.(*analysis.SyntaxError); ok {
			return nil, convertSyntaxError(statement, syntaxErr)
		}
		if err == analysis.ErrMixUserSystemTables {
			return nil, base.MixUserSystemTablesError
		}
		return nil, err
	}
	return &base.QuerySpan{
		Type:                   convertQueryType(span.Type),
		ExplainAnalyze:         span.ExplainAnalyze,
		SourceColumns:          convertColumnSet(span.SourceColumns),
		Results:                convertResults(span.Results),
		PredicateColumns:       convertColumnSet(span.PredicateColumns),
		UnresolvedColumnsError: convertUnresolvedColumnsError(span.UnresolvedColumnsError),
	}, nil
}

// convertSyntaxError places the error one byte before the parser's offset, so
// an error at the end of the statement points at its last character.
func convertSyntaxError(statement string, err *analysis.SyntaxError) *base.SyntaxError {
	syntaxErr := &base.SyntaxError{
		Message:  err.Error(),
		Position: &storepb.Position{Line: 1, Column: 1},
	}
	var parseErr *omniparser.ParseError
	if errors.As(err.Err, &parseErr) && parseErr.Position > 0 {
		syntaxErr.Position = ByteOffsetToRunePosition(statement, parseErr.Position-1)
	}
	return syntaxErr
}

// newCatalog reads the instance's synced metadata for omni.
func newCatalog(ctx context.Context, gCtx base.GetQuerySpanContext) analysis.Catalog {
	return analysis.Catalog{
		GetDatabase: func(name string) (analysis.Database, error) {
			_, database, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, name)
			if err != nil {
				return nil, err
			}
			if database == nil {
				return nil, nil
			}
			return &catalogDatabase{database: database}, nil
		},
	}
}

// catalogDatabase answers omni's lookups with the metadata model's own name
// matching.
type catalogDatabase struct {
	database *model.DatabaseMetadata
}

func (d *catalogDatabase) Metadata() *metadata.DatabaseSchemaMetadata {
	return d.database.GetProto()
}

func (d *catalogDatabase) GetTable(schema, table string) *metadata.TableMetadata {
	tableMetadata := d.database.GetSchemaMetadata(schema).GetTable(table)
	if tableMetadata == nil {
		return nil
	}
	return tableMetadata.GetProto()
}

func (d *catalogDatabase) SearchRelation(searchPath []string, name string) (string, string) {
	return d.database.SearchRelation(searchPath, name)
}

func (d *catalogDatabase) SearchFunctions(searchPath []string, name string) []*metadata.FunctionMetadata {
	_, functions := d.database.SearchFunctions(searchPath, name)
	return functions
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

func convertColumnResource(column analysis.ColumnResource) base.ColumnResource {
	return base.ColumnResource{
		Database: column.Database,
		Schema:   column.Schema,
		Table:    column.Table,
		Column:   column.Column,
	}
}

func convertColumnSet(set analysis.ColumnSet) base.SourceColumnSet {
	if set == nil {
		return nil
	}
	result := make(base.SourceColumnSet, len(set))
	for column := range set {
		result[convertColumnResource(column)] = true
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

func convertUnresolvedColumnsError(err *analysis.UnresolvedColumnsError) *base.UnresolvedColumnsError {
	if err == nil {
		return nil
	}
	var relations []base.ColumnResource
	for _, relation := range err.Relations {
		relations = append(relations, convertColumnResource(relation))
	}
	return &base.UnresolvedColumnsError{Relations: relations}
}
