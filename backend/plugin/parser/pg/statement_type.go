package pg

import (
	"strings"

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
		omniAST, ok := unifiedAST.(*OmniAST)
		if !ok {
			return nil, errors.New("expected OmniAST for PostgreSQL")
		}
		if omniAST.Node == nil {
			continue
		}

		stmtType := classifyStatementType(omniAST.Node)
		if stmtType == storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED {
			continue
		}

		line := 0
		if omniAST.StartPosition != nil {
			line = int(omniAST.StartPosition.Line)
		}

		// Compute end line from text
		endLine := line
		if omniAST.Text != "" {
			endLine += strings.Count(omniAST.Text, "\n")
		}

		allResults = append(allResults, StatementTypeWithPosition{
			Type: stmtType,
			Line: endLine,
			Text: omniAST.Text,
		})
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
