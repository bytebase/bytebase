package v2

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_MSSQL, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(
	ctx context.Context,
	statement, database, schema string,
	getDatabaseMetadata base.GetDatabaseMetadataFunc,
	listDatabaseFunc base.ListDatabaseNamesFunc,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	q := newQuerySpanExtractor(database, schema, getDatabaseMetadata, listDatabaseFunc, ignoreCaseSensitive)
	querySpan, err := q.getQuerySpan(ctx, statement)
	if err != nil {
		return nil, err
	}
	return querySpan, nil
}
