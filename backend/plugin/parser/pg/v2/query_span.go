package v2

import (
	"context"

	pgquery "github.com/pganalyze/pg_query_go/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_POSTGRES, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_REDSHIFT, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_RISINGWAVE, GetQuerySpan)
}

// querySpanExtractor is the extractor to extract the query span from the given pgquery.RawStmt.
type querySpanExtractor struct {
	ctx         context.Context
	connectedDB string
	// The metaCache serves as a lazy-load cache for the database metadata
	// and should not be accessed directly. Instead, use querySpanExtractor.getDatabaseMetadata to access it.
	metaCache map[string]*model.DatabaseMetadata
	ast       *pgquery.RawStmt
	f         base.GetDatabaseMetadataFunc
}

// newQuerySpanExtractor creates a new query span extractor, the databaseMetadata and the ast are in the read guard.
func newQuerySpanExtractor(ast *pgquery.RawStmt, connectedDB string, getDatabaseMetadata base.GetDatabaseMetadataFunc) *querySpanExtractor {
	return &querySpanExtractor{
		connectedDB: connectedDB,
		metaCache:   make(map[string]*model.DatabaseMetadata),
		ast:         ast,
		f:           getDatabaseMetadata,
	}
}

// nolint: unused
func (q *querySpanExtractor) getDatabaseMetadata(database string) (*model.DatabaseMetadata, error) {
	if meta, ok := q.metaCache[database]; ok {
		return meta, nil
	}
	meta, err := q.f(q.ctx, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", database)
	}
	q.metaCache[database] = meta
	return meta, nil
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context) (*base.QuerySpan, error) {
	q.ctx = ctx
	return &base.QuerySpan{}, nil
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, statement, database string, getDatabaseMetadata base.GetDatabaseMetadataFunc) (*base.QuerySpan, error) {
	res, err := pgquery.Parse(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse statement: %s", statement)
	}
	if len(res.Stmts) != 1 {
		return nil, errors.Errorf("expecting 1 statement, but got %d", len(res.Stmts))
	}
	ast := res.Stmts[0]
	extractor := newQuerySpanExtractor(ast, database, getDatabaseMetadata)

	switch ast.Stmt.Node.(type) {
	case *pgquery.Node_SelectStmt:
	case *pgquery.Node_ExplainStmt:
		// Skip the EXPLAIN statement.
		return &base.QuerySpan{}, nil
	default:
		return nil, errors.Wrapf(err, "expect a query statement but found %T", ast.Stmt.Node)
	}

	span, err := extractor.getQuerySpan(ctx)
	// TODO(zp): handle query system schema.
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", statement)
	}
	return span, nil
}
