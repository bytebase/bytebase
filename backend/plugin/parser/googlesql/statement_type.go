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
// AST: base.ExtractASTs keeps only non-nil ASTs, so no AST means the statement
// disappears from classification instead of reporting UNSPECIFIED.
func ParseStatements(statement string, cfg Config) ([]base.ParsedStatement, error) {
	stmts, err := SplitSQL(statement, cfg)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		file, err := parser.Parse(stmt.Text)
		if err != nil || file == nil || len(file.Stmts) == 0 {
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST:       &AST{StartPosition: stmt.Start},
			})
			continue
		}
		for _, node := range file.Stmts {
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST:       &AST{Node: node, StartPosition: stmt.Start},
			})
		}
	}
	return result, nil
}

// GetStatementTypes maps each parsed GoogleSQL statement onto the storepb
// statement type that approval rules and risk levels are written against.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	types := make([]storepb.StatementType, 0, len(asts))
	for _, a := range asts {
		node, ok := a.(*AST)
		if !ok || node.Node == nil {
			types = append(types, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED)
			continue
		}
		types = append(types, convertStatementType(analysis.StatementTypeOf(node.Node)))
	}
	return types, nil
}

func convertStatementType(t analysis.StatementType) storepb.StatementType {
	switch t {
	case analysis.Insert:
		return storepb.StatementType_INSERT
	case analysis.Update:
		return storepb.StatementType_UPDATE
	case analysis.Delete:
		return storepb.StatementType_DELETE
	case analysis.Merge:
		return storepb.StatementType_MERGE
	case analysis.Truncate:
		return storepb.StatementType_TRUNCATE
	case analysis.CreateTable:
		return storepb.StatementType_CREATE_TABLE
	case analysis.CreateView:
		return storepb.StatementType_CREATE_VIEW
	case analysis.CreateIndex:
		return storepb.StatementType_CREATE_INDEX
	case analysis.CreateSchema:
		return storepb.StatementType_CREATE_SCHEMA
	case analysis.CreateDatabase:
		return storepb.StatementType_CREATE_DATABASE
	case analysis.CreateFunction:
		return storepb.StatementType_CREATE_FUNCTION
	case analysis.CreateProcedure:
		return storepb.StatementType_CREATE_PROCEDURE
	case analysis.CreateSequence:
		return storepb.StatementType_CREATE_SEQUENCE
	case analysis.AlterTable:
		return storepb.StatementType_ALTER_TABLE
	case analysis.AlterView:
		return storepb.StatementType_ALTER_VIEW
	case analysis.AlterIndex:
		return storepb.StatementType_ALTER_INDEX
	case analysis.AlterDatabase:
		return storepb.StatementType_ALTER_DATABASE
	case analysis.AlterSequence:
		return storepb.StatementType_ALTER_SEQUENCE
	case analysis.DropTable:
		return storepb.StatementType_DROP_TABLE
	case analysis.DropView:
		return storepb.StatementType_DROP_VIEW
	case analysis.DropIndex:
		return storepb.StatementType_DROP_INDEX
	case analysis.DropSchema:
		return storepb.StatementType_DROP_SCHEMA
	case analysis.DropDatabase:
		return storepb.StatementType_DROP_DATABASE
	case analysis.DropSequence:
		return storepb.StatementType_DROP_SEQUENCE
	case analysis.DropFunction:
		return storepb.StatementType_DROP_FUNCTION
	case analysis.DropProcedure:
		return storepb.StatementType_DROP_PROCEDURE
	case analysis.Rename:
		return storepb.StatementType_RENAME
	case analysis.Execute:
		return storepb.StatementType_EXECUTE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}
