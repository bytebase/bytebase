package cassandra

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_CASSANDRA, GetQuerySpan)
}

func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	stmts, err := ParseCQL(stmt.Text)
	if err != nil {
		return nil, convertOmniError(err, base.Statement{Text: stmt.Text})
	}
	if len(stmts) == 0 {
		return &base.QuerySpan{
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}
	if len(stmts) != 1 {
		return nil, errors.Errorf("expecting only one statement to get query span, but got %d", len(stmts))
	}

	extractor := newQuerySpanExtractor(ctx, database, gCtx)
	return extractor.extract(stmts[0].AST), nil
}
