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
		return false, false, convertParseError(statement, err, nil)
	}
	return false, false, nil
}

func convertParseError(statement string, err error, basePos *storepb.Position) error {
	return base.ConvertParseError(statement, err, basePos, func(pe *parser.ParseError) (int, string, string) {
		return pe.Position, pe.Error(), pe.Message
	})
}
