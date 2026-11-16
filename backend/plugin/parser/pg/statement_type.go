package pg

import (
	"strings"

	"github.com/bytebase/omni/pg/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// StatementTypeWithPosition contains statement type and its position information.
type StatementTypeWithPosition struct {
	Type storepb.StatementType
	// Line is the one-based line number where the statement ends.
	Line int
	Text string
}

// GetStatementTypes returns statement types with position information.
// The line numbers are one-based.
func GetStatementTypes(asts []base.AST) ([]StatementTypeWithPosition, error) {
	if len(asts) == 0 {
		return []StatementTypeWithPosition{}, nil
	}

	var allResults []StatementTypeWithPosition
	for _, unifiedAST := range asts {
		wrapped, ok := unifiedAST.(*AST)
		if !ok {
			return nil, errors.New("expected AST for PostgreSQL")
		}
		if wrapped.Node == nil {
			continue
		}

		stmtTypes := convertStatementTypes(analysis.StatementTypes(wrapped.Node))
		if len(stmtTypes) == 0 {
			continue
		}

		line := 0
		if wrapped.StartPosition != nil {
			line = int(wrapped.StartPosition.Line)
		}

		// Compute end line from text
		endLine := line
		if wrapped.Text != "" {
			endLine += strings.Count(wrapped.Text, "\n")
		}

		for _, stmtType := range stmtTypes {
			allResults = append(allResults, StatementTypeWithPosition{
				Type: stmtType,
				Line: endLine,
				Text: wrapped.Text,
			})
		}
	}

	return allResults, nil
}

// GetStatementTypesForRegistry returns only the statement types.
// This is used for registration with base.RegisterGetStatementTypes.
func GetStatementTypesForRegistry(asts []base.AST) ([]storepb.StatementType, error) {
	results, err := GetStatementTypes(asts)
	if err != nil {
		return nil, err
	}
	types := make([]storepb.StatementType, len(results))
	for i, r := range results {
		types[i] = r.Type
	}
	return types, nil
}

// convertStatementTypes maps omni's statement types to storepb's by name and
// drops a name storepb does not know.
func convertStatementTypes(types []analysis.StatementType) []storepb.StatementType {
	var result []storepb.StatementType
	for _, t := range types {
		if value := storepb.StatementType(storepb.StatementType_value[string(t)]); value != storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED {
			result = append(result, value)
		}
	}
	return result
}
