package partiql

import (
	"errors"

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
		return false, false, convertParseError(statement, parseErr, nil)
	}
	return false, false, nil
}

// convertParseError converts an omni *parser.ParseError to a
// *base.SyntaxError that sql_service.go recognizes for structured
// error diagnostics. It converts the byte offset in ParseError.Position
// to 1-based line and 1-based column (rune-based) matching the
// storepb.Position convention used across all omni parser adapters.
//
// If basePos is non-nil, the computed position is offset by it so that
// errors from parsing an isolated statement segment are reported in the
// coordinates of the original multi-statement script.
func convertParseError(statement string, pe *parser.ParseError, basePos *storepb.Position) *base.SyntaxError {
	return &base.SyntaxError{
		Position:   base.ShiftPosition(base.ByteOffsetToRunePosition(statement, pe.Position), basePos),
		Message:    pe.Error(),
		RawMessage: pe.Message,
	}
}
