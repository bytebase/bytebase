package googlesql

import (
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
		file, errs := parser.Parse(stmt.Text)
		if len(errs) > 0 || file == nil || len(file.Stmts) == 0 {
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
		types = append(types, statementType(node.Node))
	}
	return types, nil
}

// statementType maps one omni GoogleSQL node to a storepb.StatementType.
//
// Deferral: every GoogleSQL statement the enum has no value for reports
// UNSPECIFIED, so a rule selecting DML skips it. The gap includes statements
// that do write rows — LOAD DATA INTO, CLONE DATA INTO, scripting bodies
// (BEGIN…END, CALL, FOR, WHILE, each of which parses as one statement), and
// EXECUTE IMMEDIATE, whose text is only known at run time. It also includes
// ALTER SCHEMA, row access policies, generic entities, and the Spanner-only
// DDL set (change streams, roles, GRANT / REVOKE, proto bundles), which do
// not write rows.
//
// A new row-writing statement needs its own enum value before it lands here.
// analysis.ClassifySQL returns Unknown for these too, so it cannot gate the list.
func statementType(node ast.Node) storepb.StatementType {
	switch n := node.(type) {
	case *ast.InsertStmt:
		return storepb.StatementType_INSERT
	case *ast.UpdateStmt:
		return storepb.StatementType_UPDATE
	case *ast.DeleteStmt:
		return storepb.StatementType_DELETE
	case *ast.MergeStmt:
		return storepb.StatementType_MERGE
	case *ast.TruncateStmt:
		return storepb.StatementType_TRUNCATE

	case *ast.CreateTableStmt:
		return storepb.StatementType_CREATE_TABLE
	case *ast.CreateViewStmt:
		return storepb.StatementType_CREATE_VIEW
	case *ast.CreateMaterializedViewStmt:
		return storepb.StatementType_CREATE_VIEW
	case *ast.CreateIndexStmt:
		return storepb.StatementType_CREATE_INDEX
	case *ast.SearchVectorIndexStmt:
		return storepb.StatementType_CREATE_INDEX
	case *ast.CreateSchemaStmt:
		return storepb.StatementType_CREATE_SCHEMA
	case *ast.CreateDatabaseStmt:
		return storepb.StatementType_CREATE_DATABASE
	case *ast.CreateFunctionStmt:
		return storepb.StatementType_CREATE_FUNCTION
	case *ast.CreateProcedureStmt:
		return storepb.StatementType_CREATE_PROCEDURE
	case *ast.CreateSequenceStmt:
		return storepb.StatementType_CREATE_SEQUENCE
	case *ast.CreateSnapshotStmt:
		return storepb.StatementType_CREATE_TABLE

	case *ast.AlterStmt:
		return alterStatementType(n.Object)
	case *ast.DropStmt:
		return dropStatementType(n.Object)
	case *ast.AlterSequenceStmt:
		return storepb.StatementType_ALTER_SEQUENCE
	case *ast.DropSequenceStmt:
		return storepb.StatementType_DROP_SEQUENCE
	case *ast.BQAlterStmt:
		return bqAlterStatementType(n.Object)
	case *ast.BQDropStmt:
		return bqDropStatementType(n.Object)

	case *ast.RenameStmt:
		return storepb.StatementType_RENAME

	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

func alterStatementType(kind ast.AlterObjectKind) storepb.StatementType {
	switch kind {
	case ast.AlterTable:
		return storepb.StatementType_ALTER_TABLE
	case ast.AlterView:
		return storepb.StatementType_ALTER_VIEW
	case ast.AlterIndex, ast.AlterSearchIndex:
		return storepb.StatementType_ALTER_INDEX
	case ast.AlterDatabase:
		return storepb.StatementType_ALTER_DATABASE
	default:
		// ALTER SCHEMA has no enum value; see the deferral on statementType.
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

func dropStatementType(kind ast.DropObjectKind) storepb.StatementType {
	switch kind {
	case ast.DropTable:
		return storepb.StatementType_DROP_TABLE
	case ast.DropView:
		return storepb.StatementType_DROP_VIEW
	case ast.DropIndex:
		return storepb.StatementType_DROP_INDEX
	case ast.DropSchema:
		return storepb.StatementType_DROP_SCHEMA
	case ast.DropDatabase:
		return storepb.StatementType_DROP_DATABASE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

func bqAlterStatementType(kind ast.BQAlterObjectKind) storepb.StatementType {
	switch kind {
	case ast.BQAlterMaterializedView, ast.BQAlterApproxView:
		return storepb.StatementType_ALTER_VIEW
	case ast.BQAlterVectorIndex:
		return storepb.StatementType_ALTER_INDEX
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

func bqDropStatementType(kind ast.BQDropObjectKind) storepb.StatementType {
	switch kind {
	case ast.BQDropFunction, ast.BQDropTableFunction:
		return storepb.StatementType_DROP_FUNCTION
	case ast.BQDropProcedure:
		return storepb.StatementType_DROP_PROCEDURE
	case ast.BQDropMaterializedView, ast.BQDropApproxView:
		return storepb.StatementType_DROP_VIEW
	case ast.BQDropSnapshotTable:
		return storepb.StatementType_DROP_TABLE
	case ast.BQDropSearchIndex, ast.BQDropVectorIndex:
		return storepb.StatementType_DROP_INDEX
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}
