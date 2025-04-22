package trino

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_TRINO, GetQuerySpan)
}

// GetQuerySpan gets the query span for Trino.
// This is the entry point registered with the base package.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, statement, database, schema string, ignoreCaseSensitive bool) (*base.QuerySpan, error) {
	extractor := newQuerySpanExtractor(database, schema, gCtx, ignoreCaseSensitive)

	querySpan, err := extractor.getQuerySpan(ctx, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", statement)
	}

	return querySpan, nil
}
