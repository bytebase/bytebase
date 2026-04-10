package elasticsearch

import (
	"context"

	lsp "github.com/bytebase/lsp-protocol"

	es "github.com/bytebase/omni/elasticsearch"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_ELASTICSEARCH, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	omniRanges, err := es.GetStatementRanges(statement)
	if err != nil {
		return nil, err
	}
	if omniRanges == nil {
		return []base.Range{}, nil
	}

	ranges := make([]base.Range, 0, len(omniRanges))
	for _, r := range omniRanges {
		ranges = append(ranges, base.Range{
			Start: lsp.Position{
				Line:      r.Start.Line,
				Character: r.Start.Character,
			},
			End: lsp.Position{
				Line:      r.End.Line,
				Character: r.End.Character,
			},
		})
	}
	return ranges, nil
}
