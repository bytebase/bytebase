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
	meta, err := base.RequireDatabaseMetadata(ctx, gCtx, database)
	if err != nil {
		return nil, err
	}
	searchPath := meta.GetSearchPath()
	if schema != "" {
		searchPath = analysis.SearchPathForSelectedSchema(schema, searchPath)
	}

	span, err := analysis.ResolveQuerySpan(ctx, stmt.Text, analysis.ResolveOptions{
		DefaultDatabase: database,
		SearchPath:      searchPath,
		Catalog: analysis.Catalog{
			GetDatabase: base.QuerySpanDatabaseGetter(ctx, gCtx, false, func(db *model.DatabaseMetadata) analysis.Database {
				return &catalogDatabase{database: db}
			}),
		},
	})
	if syntaxErr, ok := err.(*analysis.SyntaxError); ok {
		err = convertSyntaxError(stmt.Text, syntaxErr)
	}
	querySpan, err := base.OmniResolvedSpanResult(span, err)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
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
