// Package mysql implements the SQL advisor rules for MySQL.
package mysql

import (
	"strings"

	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
)

// contentStartLine returns the 1-based line number of the first non-whitespace
// character in text. Returns 1 if the text is empty or has no leading newlines.
func contentStartLine(text string) int32 {
	idx := contentStartIndex(text)
	if idx <= 0 {
		return 1
	}
	return mysqlparser.ByteOffsetToRunePosition(text, idx).Line
}

// contentEndLine returns the 1-based line number of the last non-whitespace
// character in text.
func contentEndLine(text string) int32 {
	if text == "" {
		return 1
	}
	idx := -1
	for i := len(text) - 1; i >= 0; i-- {
		c := text[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != ';' {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return 1
	}
	return mysqlparser.ByteOffsetToRunePosition(text, idx).Line
}

// findLineByName searches for an identifier name in text and returns its
// 1-based line number. Falls back to contentStartLine(text) if not found.
func findLineByName(text, name string) int32 {
	if name == "" || text == "" {
		return contentStartLine(text)
	}
	idx := strings.Index(text, name)
	if idx < 0 {
		return contentStartLine(text)
	}
	return mysqlparser.ByteOffsetToRunePosition(text, idx).Line
}

// trimmedStmtText returns text with leading/trailing whitespace and trailing
// semicolons removed.
func trimmedStmtText(text string) string {
	return strings.TrimRight(strings.TrimSpace(text), ";")
}

// columnFindingPosition is where a finding's advice starts: the statement's first
// line plus the line of the finding's anchor within it.
func columnFindingPosition(stmt base.ParsedStatement, f v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f)))
}

// statementPosition is where advice about a whole statement starts: the line
// of the statement's first non-blank character.
func statementPosition(stmt base.ParsedStatement) *storepb.Position {
	return columnFindingPosition(stmt, v1review.Finding{})
}

// v1reviewSchema returns the snapshot md holds for an omni rule, an empty
// one when md is nil.
func v1reviewSchema(md *model.DatabaseMetadata) v1review.Schema {
	if md == nil {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: md.GetProto(), ObjectCaseSensitive: md.GetIsObjectCaseSensitive()}
}
