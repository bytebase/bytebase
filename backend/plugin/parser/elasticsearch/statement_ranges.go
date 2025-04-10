package elasticsearch

import (
	"context"
	"unicode/utf8"

	lsp "github.com/bytebase/lsp-protocol"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_ELASTICSEARCH, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	parseResult, _ := ParseElasticsearchREST(statement)
	if parseResult == nil {
		return []base.Range{}, nil
	}
	bs := []byte(statement)
	var ranges []base.Range
	for _, r := range parseResult.Requests {
		if r == nil {
			continue
		}
		if r.EndOffset <= r.StartOffset {
			continue
		}

		// Get the start and end position of the request.
		startPosition := getPositionByByteOffset(r.StartOffset, bs)
		endPosition := getPositionByByteOffset(r.EndOffset, bs)
		if startPosition == nil || endPosition == nil {
			continue
		}
		ranges = append(ranges, base.Range{
			Start: *startPosition,
			End:   *endPosition,
		})
	}
	return ranges, nil
}

func getPositionByByteOffset(byteOffset int, bs []byte) *lsp.Position {
	var position lsp.Position
	for i := 0; ; {
		if i >= byteOffset || i > len(bs) {
			break
		}
		if bs[i] == '\n' {
			position.Line++
			position.Character = 0
			i++
			continue
		}
		r, size := utf8.DecodeRune(bs[i:])
		if r == utf8.RuneError {
			return nil
		}
		position.Character++
		if r > 0xFFFF {
			// Out of BMP, need surrogate pair.
			position.Character++
		}
		i += size
	}

	return &position
}
