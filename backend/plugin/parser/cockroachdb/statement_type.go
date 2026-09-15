package cockroachdb

import (
	"slices"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetStatementTypes(storepb.Engine_COCKROACHDB, GetStatementTypes)
}

// GetStatementTypes returns the types of the statements that have one, in order, each followed by
// the types of the mutations in its CTEs and statement sources, each type once per statement.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	var statementTypes []storepb.StatementType
	for _, ast := range asts {
		crdbAST, ok := ast.(*AST)
		if !ok {
			return nil, errors.New("expected CockroachDB AST")
		}
		var types []storepb.StatementType
		add := func(statementType storepb.StatementType) {
			if statementType != storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED && !slices.Contains(types, statementType) {
				types = append(types, statementType)
			}
		}
		stmt := crdbAST.Stmt.AST
		// EXPLAIN ANALYZE executes the statement it explains.
		if explain, ok := stmt.(*tree.ExplainAnalyze); ok {
			stmt = explain.Statement
		}
		add(getStatementType(stmt))
		if createTable, ok := stmt.(*tree.CreateTable); ok && createTable.AsSource != nil {
			stmt = createTable.AsSource
		}
		for _, mutationType := range getMutationTypes(stmt) {
			add(mutationType)
		}
		statementTypes = append(statementTypes, types...)
	}
	return statementTypes, nil
}

func getStatementType(stmt tree.Statement) storepb.StatementType {
	switch n := stmt.(type) {
	case *tree.CreateDatabase:
		return storepb.StatementType_CREATE_DATABASE
	case *tree.CreateTable:
		return storepb.StatementType_CREATE_TABLE
	case *tree.CreateView:
		return storepb.StatementType_CREATE_VIEW
	case *tree.CreateIndex:
		return storepb.StatementType_CREATE_INDEX
	case *tree.CreateSequence:
		return storepb.StatementType_CREATE_SEQUENCE
	case *tree.CreateSchema:
		return storepb.StatementType_CREATE_SCHEMA
	case *tree.CreateRoutine:
		// Procedures are functions here, as PostgreSQL classifies CREATE PROCEDURE.
		return storepb.StatementType_CREATE_FUNCTION
	case *tree.CreateTrigger:
		return storepb.StatementType_CREATE_TRIGGER
	case *tree.CreateExtension:
		return storepb.StatementType_CREATE_EXTENSION
	case *tree.CreateType:
		return storepb.StatementType_CREATE_TYPE

	case *tree.DropDatabase:
		return storepb.StatementType_DROP_DATABASE
	case *tree.DropTable:
		return storepb.StatementType_DROP_TABLE
	case *tree.DropView:
		// A materialized view holds data, so dropping one is as risky as dropping a table.
		if n.IsMaterialized {
			return storepb.StatementType_DROP_TABLE
		}
		return storepb.StatementType_DROP_VIEW
	case *tree.DropIndex:
		return storepb.StatementType_DROP_INDEX
	case *tree.DropSequence:
		return storepb.StatementType_DROP_SEQUENCE
	case *tree.DropSchema:
		return storepb.StatementType_DROP_SCHEMA
	case *tree.DropType:
		return storepb.StatementType_DROP_TYPE
	case *tree.DropTrigger:
		return storepb.StatementType_DROP_TRIGGER
	case *tree.DropRoutine:
		return storepb.StatementType_DROP_FUNCTION
	case *tree.Truncate:
		return storepb.StatementType_TRUNCATE

	case *tree.RenameDatabase, *tree.AlterDatabaseOwner, *tree.ReparentDatabase,
		*tree.AlterDatabaseAddRegion, *tree.AlterDatabaseDropRegion, *tree.AlterDatabasePrimaryRegion,
		*tree.AlterDatabaseSurvivalGoal, *tree.AlterDatabasePlacement, *tree.AlterDatabaseAddSuperRegion,
		*tree.AlterDatabaseDropSuperRegion, *tree.AlterDatabaseAlterSuperRegion, *tree.AlterDatabaseSecondaryRegion,
		*tree.AlterDatabaseDropSecondaryRegion, *tree.AlterDatabaseSetZoneConfigExtension:
		return storepb.StatementType_ALTER_DATABASE
	case *tree.SetZoneConfig:
		switch {
		case n.Database != "":
			return storepb.StatementType_ALTER_DATABASE
		case n.TableOrIndex.Index != "":
			return storepb.StatementType_ALTER_INDEX
		case n.TableOrIndex.Table.ObjectName != "":
			return storepb.StatementType_ALTER_TABLE
		default:
			return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
		}
	case *tree.AlterRoleSet:
		// ALTER DATABASE d SET parses as ALTER ROLE ALL IN DATABASE d SET, which sets the same defaults.
		if n.AllRoles && n.DatabaseName != "" {
			return storepb.StatementType_ALTER_DATABASE
		}
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	case *tree.AlterTable, *tree.AlterTableLocality:
		return storepb.StatementType_ALTER_TABLE
	case *tree.AlterTableSetSchema:
		return getRelationAlterType(n.IsView && !n.IsMaterialized, n.IsSequence)
	case *tree.AlterTableOwner:
		return getRelationAlterType(n.IsView && !n.IsMaterialized, n.IsSequence)
	case *tree.AlterIndex, *tree.AlterIndexVisible:
		return storepb.StatementType_ALTER_INDEX
	case *tree.AlterSequence:
		return storepb.StatementType_ALTER_SEQUENCE
	case *tree.AlterType:
		return storepb.StatementType_ALTER_TYPE

	case *tree.RenameTable:
		switch {
		case n.IsSequence:
			return storepb.StatementType_RENAME_SEQUENCE
		case n.IsView && !n.IsMaterialized:
			return storepb.StatementType_ALTER_VIEW
		default:
			return storepb.StatementType_ALTER_TABLE
		}
	case *tree.RenameIndex:
		return storepb.StatementType_RENAME_INDEX
	case *tree.AlterSchema:
		if _, ok := n.Cmd.(*tree.AlterSchemaRename); ok {
			return storepb.StatementType_RENAME_SCHEMA
		}
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED

	case *tree.CommentOnColumn, *tree.CommentOnConstraint, *tree.CommentOnDatabase, *tree.CommentOnIndex,
		*tree.CommentOnSchema, *tree.CommentOnTable, *tree.CommentOnType:
		return storepb.StatementType_COMMENT

	case *tree.Insert:
		return storepb.StatementType_INSERT
	case *tree.Update:
		return storepb.StatementType_UPDATE
	case *tree.Delete:
		return storepb.StatementType_DELETE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

func getRelationAlterType(isView bool, isSequence bool) storepb.StatementType {
	switch {
	case isSequence:
		return storepb.StatementType_ALTER_SEQUENCE
	case isView:
		return storepb.StatementType_ALTER_VIEW
	default:
		return storepb.StatementType_ALTER_TABLE
	}
}
