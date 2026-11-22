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
		types = append(types, statementTypeMap[analysis.StatementTypeOf(node.Node)])
	}
	return types, nil
}

var statementTypeMap = map[analysis.StatementType]storepb.StatementType{
	analysis.Insert:          storepb.StatementType_INSERT,
	analysis.Update:          storepb.StatementType_UPDATE,
	analysis.Delete:          storepb.StatementType_DELETE,
	analysis.Merge:           storepb.StatementType_MERGE,
	analysis.Truncate:        storepb.StatementType_TRUNCATE,
	analysis.CreateTable:     storepb.StatementType_CREATE_TABLE,
	analysis.CreateView:      storepb.StatementType_CREATE_VIEW,
	analysis.CreateIndex:     storepb.StatementType_CREATE_INDEX,
	analysis.CreateSchema:    storepb.StatementType_CREATE_SCHEMA,
	analysis.CreateDatabase:  storepb.StatementType_CREATE_DATABASE,
	analysis.CreateFunction:  storepb.StatementType_CREATE_FUNCTION,
	analysis.CreateProcedure: storepb.StatementType_CREATE_PROCEDURE,
	analysis.CreateSequence:  storepb.StatementType_CREATE_SEQUENCE,
	analysis.AlterTable:      storepb.StatementType_ALTER_TABLE,
	analysis.AlterView:       storepb.StatementType_ALTER_VIEW,
	analysis.AlterIndex:      storepb.StatementType_ALTER_INDEX,
	analysis.AlterDatabase:   storepb.StatementType_ALTER_DATABASE,
	analysis.AlterSequence:   storepb.StatementType_ALTER_SEQUENCE,
	analysis.DropTable:       storepb.StatementType_DROP_TABLE,
	analysis.DropView:        storepb.StatementType_DROP_VIEW,
	analysis.DropIndex:       storepb.StatementType_DROP_INDEX,
	analysis.DropSchema:      storepb.StatementType_DROP_SCHEMA,
	analysis.DropDatabase:    storepb.StatementType_DROP_DATABASE,
	analysis.DropSequence:    storepb.StatementType_DROP_SEQUENCE,
	analysis.DropFunction:    storepb.StatementType_DROP_FUNCTION,
	analysis.DropProcedure:   storepb.StatementType_DROP_PROCEDURE,
	analysis.Rename:          storepb.StatementType_RENAME,
	analysis.Execute:         storepb.StatementType_EXECUTE,
}
