package pg

import (
	"context"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/bytebase/omni/pg/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_POSTGRES, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	var ranges []base.Range
	for _, r := range analysis.StatementRanges(statement) {
		ranges = append(ranges, lsp.Range{
			Start: lsp.Position{Line: r.Start.Line, Character: r.Start.Character},
			End:   lsp.Position{Line: r.End.Line, Character: r.End.Character},
		})
	}
	return ranges, nil
}
