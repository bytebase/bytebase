package v2

import (
	"context"

	pgquery "github.com/pganalyze/pg_query_go/v4"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	// defaultSchemaName is the default schema name in PostgreSQL like DBMS.
	defaultSchemaName = "public"
	// unknownFieldName is the default field name for unknown field in PostgreSQL like DBMS.
	unknownFieldName = "?column?"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_POSTGRES, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_REDSHIFT, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_RISINGWAVE, GetQuerySpan)
}

// querySpanExtractor is the extractor to extract the query span from the given pgquery.RawStmt.
type querySpanExtractor struct {
	databaseMetadata *model.DatabaseMetadata
	ast              *pgquery.RawStmt
}

// newQuerySpanExtractor creates a new query span extractor, the databaseMetadata and the ast are in the read guard.
func newQuerySpanExtractor(databaseMetadata *model.DatabaseMetadata, ast *pgquery.RawStmt) *querySpanExtractor {
	return &querySpanExtractor{
		databaseMetadata: databaseMetadata,
		ast:              ast,
	}
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, statement string, getDatabaseMetadata base.GetDatabaseMetadataFunc) (*base.QuerySpan, error) {
	return nil, nil
}
