package plsql

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_ORACLE, GetQuerySpan)
}

func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	extractor := newQuerySpanExtractor(database, gCtx)

	querySpan, err := extractor.getQuerySpan(ctx, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", statement)
	}
	return querySpan, nil
}
