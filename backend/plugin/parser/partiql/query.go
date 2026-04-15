package partiql

import (
	"errors"
	"unicode/utf8"

	"github.com/bytebase/omni/partiql/analysis"
	"github.com/bytebase/omni/partiql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_DYNAMODB, validateQuery)
}

func validateQuery(statement string) (bool, bool, error) {
	err := analysis.ValidateQuery(statement)
	if err == nil {
		return true, true, nil
	}
	// Distinguish syntax errors from validation rejections.
	// Parse errors (*parser.ParseError) are converted to
	// *base.SyntaxError so callers (e.g. sql_service.go's
	// validateQueryRequest) get structured line/column diagnostics
	// via errors.As(err, &parserbase.SyntaxError{}).
	// Validation errors (DML/DDL/EXEC rejected) mean the query is
	// syntactically valid but not read-only.
	var parseErr *parser.ParseError
	if errors.As(err, &parseErr) {
		return false, false, convertParseError(statement, parseErr)
	}
	return false, false, nil
}

// convertParseError converts an omni *parser.ParseError to a
// *base.SyntaxError that sql_service.go recognizes for structured
// error diagnostics. It converts the byte offset in ParseError.Loc
// to 1-based line and 1-based column (rune-based) matching the
// storepb.Position convention used across all omni parser adapters.
func convertParseError(statement string, pe *parser.ParseError) *base.SyntaxError {
	line, col := byteOffsetToPosition(statement, pe.Loc.Start)
	return &base.SyntaxError{
		Position: &storepb.Position{
			Line:   int32(line),
			Column: int32(col),
		},
		Message:    pe.Error(),
		RawMessage: pe.Message,
	}
}

// byteOffsetToPosition converts a 0-based byte offset to a 1-based
// line number and 1-based column (in runes). Both line and column are
// 1-based to match the storepb.Position convention used by other omni
// parser adapters (see e.g. tsql/omni.go: "runeCol + 1").
func byteOffsetToPosition(s string, offset int) (line, col int) {
	line = 1
	col = 1
	for i := 0; i < offset && i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\n' {
			line++
			col = 1
		} else {
			col++
		}
		i += size
	}
	return line, col
}
