package trino

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_TRINO, GetQuerySpan)
}

// GetQuerySpan gets the query span for Trino.
// This is the entry point registered with the base package.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, ignoreCaseSensitive bool) (*base.QuerySpan, error) {
	extractor := newQuerySpanExtractor(database, schema, gCtx, ignoreCaseSensitive)

	querySpan, err := extractor.getQuerySpan(ctx, stmt.Text)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}

	return querySpan, nil
}
