package mongodb

import (
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_MONGODB, SplitSQL)
}

// SplitSQL splits the input into multiple MongoDB statements using the parser.
// Unlike SQL-based engines that split on semicolons, MongoDB statements can be
// separated by newlines without semicolons, so we use the parser to identify
// statement boundaries.
func SplitSQL(statement string) ([]base.Statement, error) {
	parseResult := ParseMongoShell(statement)
	if parseResult == nil || len(parseResult.Statements) == 0 {
		if len(strings.TrimSpace(statement)) == 0 {
			return nil, nil
		}
		return []base.Statement{{Text: statement}}, nil
	}

	runes := []rune(statement)
	var result []base.Statement

	for _, stmt := range parseResult.Statements {
		if stmt.EndOffset <= stmt.StartOffset {
			continue
		}

		endOffset := min(stmt.EndOffset, len(runes))

		text := string(runes[stmt.StartOffset:endOffset])

		// Calculate byte offsets for Range.
		byteStart := len(string(runes[:stmt.StartOffset]))
		byteEnd := len(string(runes[:endOffset]))

		result = append(result, base.Statement{
			Text: text,
			Start: &storepb.Position{
				Line:   int32(stmt.StartLine),
				Column: int32(stmt.StartColumn + 1), // Convert 0-based column to 1-based
			},
			End: &storepb.Position{
				Line:   int32(stmt.EndLine),
				Column: int32(stmt.EndColumn + 1), // Convert 0-based exclusive end to 1-based
			},
			Range: &storepb.Range{
				Start: int32(byteStart),
				End:   int32(byteEnd),
			},
		})
	}

	return result, nil
}
