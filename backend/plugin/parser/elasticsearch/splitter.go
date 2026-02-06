package elasticsearch

import (
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_ELASTICSEARCH, SplitMultiSQL)
}

// SplitMultiSQL splits the input into individual ElasticSearch REST API requests.
func SplitMultiSQL(statement string) ([]base.Statement, error) {
	parseResult, err := ParseElasticsearchREST(statement)
	if err != nil {
		return nil, err
	}

	if parseResult == nil || len(parseResult.Requests) == 0 {
		if len(strings.TrimSpace(statement)) == 0 {
			return nil, nil
		}
		return []base.Statement{{Text: statement}}, nil
	}

	var statements []base.Statement
	for _, req := range parseResult.Requests {
		if req == nil {
			continue
		}
		text := statement[req.StartOffset:req.EndOffset]
		empty := len(strings.TrimSpace(text)) == 0

		// Calculate 1-based line and column positions
		startLine, startColumn := byteOffsetToPosition(statement, req.StartOffset)
		endLine, endColumn := byteOffsetToPosition(statement, req.EndOffset)

		statements = append(statements, base.Statement{
			Text:  text,
			Empty: empty,
			Start: &storepb.Position{
				Line:   int32(startLine),
				Column: int32(startColumn),
			},
			End: &storepb.Position{
				Line:   int32(endLine),
				Column: int32(endColumn),
			},
			Range: &storepb.Range{
				Start: int32(req.StartOffset),
				End:   int32(req.EndOffset),
			},
		})
	}
	return statements, nil
}

// byteOffsetToPosition converts a byte offset to 1-based line and column numbers.
// Column is measured in Unicode code points (runes), not bytes.
func byteOffsetToPosition(text string, byteOffset int) (line, column int) {
	line = 1
	column = 1
	currentByte := 0

	for _, r := range text {
		if currentByte >= byteOffset {
			break
		}
		if r == '\n' {
			line++
			column = 1
		} else {
			column++
		}
		currentByte += len(string(r))
	}
	return line, column
}
