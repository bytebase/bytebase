package googlesql

import (
	"github.com/bytebase/omni/googlesql/ast"
	"github.com/bytebase/omni/googlesql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST wraps one omni GoogleSQL statement node so it satisfies base.AST.
type AST struct {
	// Node is the omni AST node (e.g. *ast.InsertStmt, *ast.CreateTableStmt).
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// ParseStatements splits the input with the dialect's splitter and parses each
// statement into an omni AST node.
//
// A statement omni cannot parse carries an AST with a nil Node, so
// GetStatementTypes reports it as STATEMENT_TYPE_UNSPECIFIED. It must not be
// dropped: base.ExtractASTs keeps only non-nil ASTs, so a dropped statement
// disappears from classification entirely, and a sheet mixing one statement
// omni rejects with one it accepts would be classified from the accepted one
// alone. An approval rule naming statement.sql_type would then never see the
// rejected statement, which is the silent skip BYT-10131 reported.
//
// Empty statements still carry no AST; they are separators, not changes.
func ParseStatements(statement string, cfg Config) ([]base.ParsedStatement, error) {
	stmts, err := SplitSQL(statement, cfg)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			result = append(result, base.ParsedStatement{Statement: stmt})
			continue
		}
		file, errs := parser.Parse(stmt.Text)
		if len(errs) > 0 || file == nil || len(file.Stmts) == 0 {
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST:       &AST{Text: stmt.Text, StartPosition: stmt.Start},
			})
			continue
		}
		for _, node := range file.Stmts {
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST: &AST{
					Node:          node,
					Text:          stmt.Text,
					StartPosition: stmt.Start,
				},
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
// UNSPECIFIED. A rule excluding DML then treats it as "approve", and a rule
// selecting DML treats it as "skip", so the second direction is a gap, not a
// safe default. The gap covers statements that do write rows: LOAD DATA INTO,
// CLONE DATA INTO, and EXECUTE IMMEDIATE (whose text is only known at run
// time, so no static value can be right). It also covers ALTER SCHEMA, row
// access policies, generic entities (CAPACITY / RESERVATION / ASSIGNMENT),
// scripting bodies, and the Spanner-only DDL set (change streams, roles,
// GRANT / REVOKE, proto bundles), none of which write rows.
//
// An acceptance test against omni's analysis.ClassifySQL cannot catch the
// row-writing members: omni also classifies them Unknown, so both sides agree
// vacuously. Closing the gap needs the evaluator to treat an unclassified
// statement as indeterminate rather than as a definite non-DML value; adding
// more enum values cannot cover EXECUTE IMMEDIATE.
func statementType(node ast.Node) storepb.StatementType {
	switch n := node.(type) {
	// DML.
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

	// DDL - CREATE.
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

	// DDL - ALTER / DROP carry the object kind on the node.
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
