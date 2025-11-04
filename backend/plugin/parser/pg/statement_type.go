package pg

import (
	"github.com/pkg/errors"
)

// StatementTypeWithPosition contains statement type and its position information.
type StatementTypeWithPosition struct {
	Type string
	// Line is the one-based line number where the statement ends.
	Line int
	Text string
}

// GetStatementTypesWithPositions returns statement types with position information.
// The line numbers are one-based.
func GetStatementTypesWithPositions(asts any) ([]StatementTypeWithPosition, error) {
	parseResult, ok := asts.(*ParseResult)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T, expected *ParseResult", asts)
	}
	return GetStatementTypesWithPositionsANTLR(parseResult)
}
