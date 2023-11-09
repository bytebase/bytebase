package v2

import (
	"context"

	pgquery "github.com/pganalyze/pg_query_go/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_POSTGRES, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_REDSHIFT, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_RISINGWAVE, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, statement, database string, getDatabaseMetadata base.GetDatabaseMetadataFunc) (*base.QuerySpan, error) {
	extractor := newQuerySpanExtractor(database, getDatabaseMetadata)

	res, err := pgquery.Parse(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse statement: %s", statement)
	}
	if len(res.Stmts) != 1 {
		return nil, errors.Errorf("expecting 1 statement, but got %d", len(res.Stmts))
	}
	ast := res.Stmts[0]

	switch ast.Stmt.Node.(type) {
	case *pgquery.Node_SelectStmt:
	case *pgquery.Node_ExplainStmt:
		// Skip the EXPLAIN statement.
		return &base.QuerySpan{}, nil
	default:
		return nil, errors.Wrapf(err, "expect a query statement but found %T", ast.Stmt.Node)
	}

	// Our querySpanExtractor is based on the pg_query_go library, which does not support listening to or walking the AST.
	// We separate the logic for querying spans and accessing data.
	// The second one is achieved using ParseToJson, which is simpler.
	querySpanResults, err := extractor.getQuerySpanResult(ctx, ast)
	// TODO(zp): handle query system schema.
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", statement)
	}
	return &base.QuerySpan{
		Results: querySpanResults,
	}, nil
}
