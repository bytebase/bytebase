package mariadb

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_MARIADB, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	stmt base.Statement,
	database, _ string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	q := newQuerySpanExtractor(database, gCtx, ignoreCaseSensitive)
	querySpan, err := q.getQuerySpan(ctx, stmt.Text)
	if err != nil {
		return nil, convertError(err, stmt)
	}
	return querySpan, nil
}
