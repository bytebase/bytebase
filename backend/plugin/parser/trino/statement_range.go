package trino

import (
	"context"
	"strings"

	lsp "github.com/bytebase/lsp-protocol"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_TRINO, GetStatementRanges)
}

// GetStatementRanges gets ranges for the statements in the SQL.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	// Split the SQL statement into individual statements
	sqlStmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	// Create ranges for each statement
	var ranges []base.Range

	// Track the current position
	var offset int

	for _, sql := range sqlStmts {
		if sql.Empty {
			continue
		}

		// Calculate start and end line/column for each statement
		startLine, startChar := countLineCol(statement[:offset])
		endLine, endChar := countLineCol(statement[:offset+len(sql.Text)])

		// Create a range for the statement
		ranges = append(ranges, base.Range{
			Start: lsp.Position{
				Line:      uint32(startLine),
				Character: uint32(startChar),
			},
			End: lsp.Position{
				Line:      uint32(endLine),
				Character: uint32(endChar),
			},
		})

		// Move to the next statement
		offset += len(sql.Text)
	}

	return ranges, nil
}

// countLineCol counts the line and column numbers for a position in text.
func countLineCol(text string) (int, int) {
	lines := strings.Split(text, "\n")
	lineCount := len(lines) - 1

	// If there are no newlines, column is just the length
	if lineCount <= 0 {
		return 0, len(text)
	}

	// Column is the length of the last line
	return lineCount, len(lines[lineCount])
}
