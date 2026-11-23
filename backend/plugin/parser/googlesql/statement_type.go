package googlesql

import (
	"github.com/bytebase/omni/googlesql/analysis"
	"github.com/bytebase/omni/googlesql/ast"
	"github.com/bytebase/omni/googlesql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST wraps one omni GoogleSQL statement node so it satisfies base.AST. Node is
// nil when omni could not parse the statement.
type AST struct {
	Node          ast.Node
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// ParseStatements splits the input with the dialect's splitter and parses each
// statement into an omni AST node.
//
// A statement omni cannot parse carries an AST with a nil Node rather than no
// AST so base.ExtractASTs preserves it for UNSPECIFIED classification.
func ParseStatements(statement string, cfg Config) ([]base.ParsedStatement, error) {
	return base.ParseMultiNodeStatements(statement, func(s string) ([]base.Statement, error) {
		return SplitSQL(s, cfg)
	}, func(stmt base.Statement) ([]base.AST, error) {
		if file, err := parser.Parse(stmt.Text); err == nil && file != nil && len(file.Stmts) > 0 {
			asts := make([]base.AST, len(file.Stmts))
			for i, node := range file.Stmts {
				asts[i] = &AST{Node: node, StartPosition: stmt.Start}
			}
			return asts, nil
		}
		return []base.AST{&AST{StartPosition: stmt.Start}}, nil
	})
}

// GetStatementTypes maps each parsed GoogleSQL statement onto the storepb
// statement type that approval rules and risk levels are written against.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	types := make([]storepb.StatementType, 0, len(asts))
	for _, a := range asts {
		node, ok := a.(*AST)
		if !ok || node == nil || node.Node == nil {
			types = append(types, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED)
			continue
		}
		types = append(types, base.OmniStatementType(analysis.StatementTypeOf(node.Node)))
	}
	return types, nil
}
