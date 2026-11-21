// Package mariadb implements the SQL advisor rules for MariaDB.
package mariadb

import (
	"slices"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

func init() {
	extract := func(checkCtx advisor.Context) ([]advisor.V1Stmt[ast.Node], error) {
		return advisor.ExtractNodes(checkCtx.ParsedStatements, mariadbparser.GetNode), nil
	}
	byteOffsetLine := func(sql string, offset int) int32 {
		return mariadbparser.ByteOffsetToRunePosition(sql, offset).Line
	}
	makePriorBackupStmt := func(node ast.Node, text string) mariadbv1review.PriorBackupStatement {
		return mariadbv1review.PriorBackupStatement{Node: node, Text: text}
	}
	f := advisor.NewMySQLFamily(advisor.MySQLFamilyOptions[ast.Node]{
		Extract:         extract,
		ByteOffsetLine:  byteOffsetLine,
		DefinesColumn:   definesColumn,
		ReadsColumnType: readsColumnType,
		DropsColumn:     dropsColumn,
		IsRenameIndex:   isRenameIndex,
		IsDropDatabase:  isDropDatabase,
	})

	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK, &advisor.BuiltinWalkThroughCheckAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, advisor.NewMySQLPriorBackupAdvisor(storepb.Engine_MARIADB, extract, makePriorBackupStmt, mariadbv1review.PriorBackupCheck))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_ADVICE_ONLINE_MIGRATION, f.OnlineMigration(storepb.Engine_MARIADB, mariadbv1review.OnlineMigration))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, advisor.NewMySQLRowLimitAdvisor(extract, byteOffsetLine, mariadbv1review.StatementInsertRowLimitProbes, mariadbv1review.StatementInsertRowLimit, code.InsertTooManyRows, "inserts", true))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, advisor.NewMySQLRowLimitAdvisor(extract, byteOffsetLine, mariadbv1review.StatementAffectedRowLimitProbes, mariadbv1review.StatementAffectedRowLimit, code.StatementAffectedRowExceedsLimit, "affected", false))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, advisor.NewMySQLDMLDryRunAdvisor(extract, byteOffsetLine, mariadbv1review.StatementDMLDryRunProbes, mariadbv1review.StatementDMLDryRun))

	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, f.ColumnAutoIncrementInitialValue(mariadbv1review.ColumnAutoIncrementInitialValue))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, f.ColumnAutoIncrementMustInteger(mariadbv1review.ColumnAutoIncrementMustInteger))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, f.ColumnAutoIncrementMustUnsigned(mariadbv1review.ColumnAutoIncrementMustUnsigned))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_COMMENT, f.ColumnComment(mariadbv1review.ColumnComment))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, f.ColumnCurrentTimeCountLimit(mariadbv1review.ColumnCurrentTimeCountLimit))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE, f.ColumnDisallowChange(mariadbv1review.ColumnDisallowChange))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, f.ColumnDisallowChangeType(mariadbv1review.ColumnDisallowChangeType))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, f.ColumnDisallowChangingOrder(mariadbv1review.ColumnDisallowChangingOrder))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, f.ColumnDisallowDropInIndex(mariadbv1review.ColumnDisallowDropInIndex))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, f.ColumnDisallowSetCharset(mariadbv1review.ColumnDisallowSetCharset))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, f.ColumnMaximumCharacterLength(mariadbv1review.ColumnMaximumCharacterLength))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, f.ColumnMaximumVarcharLength(mariadbv1review.ColumnMaximumVarcharLength))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_NO_NULL, f.ColumnNoNull(mariadbv1review.ColumnNoNull))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, f.ColumnRequireDefault(mariadbv1review.ColumnRequireDefault))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_REQUIRED, f.ColumnRequired(mariadbv1review.ColumnRequired))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, f.ColumnSetDefaultForNotNull(mariadbv1review.ColumnSetDefaultForNotNull))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, f.ColumnTypeDisallowList(mariadbv1review.ColumnTypeDisallowList))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, f.DatabaseDropEmptyDatabase(mariadbv1review.DatabaseDropEmptyDatabase))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB, f.EngineMySQLUseInnoDB(mariadbv1review.EngineMySQLUseInnoDB))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, f.IndexKeyNumberLimit(mariadbv1review.IndexKeyNumberLimit))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, f.IndexNoDuplicateColumn(mariadbv1review.IndexNoDuplicateColumn))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, f.IndexPKTypeLimit(mariadbv1review.IndexPKTypeLimit))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, f.IndexTotalNumberLimit(mariadbv1review.IndexTotalNumberLimit))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, f.IndexTypeNoBlob(mariadbv1review.IndexTypeNoBlob))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, f.NamingAutoIncrementColumn(mariadbv1review.NamingAutoIncrementColumn))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_COLUMN, f.NamingColumn(mariadbv1review.NamingColumn))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_INDEX_FK, f.NamingForeignKey(mariadbv1review.NamingForeignKey))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_INDEX_IDX, f.NamingIndex(mariadbv1review.NamingIndex))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_TABLE, f.NamingTable(mariadbv1review.NamingTable))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_INDEX_UK, f.NamingUniqueKey(mariadbv1review.NamingUniqueKey))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, f.SchemaBackwardCompatibility(mariadbv1review.SchemaBackwardCompatibility))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, f.StatementDisallowCommit(mariadbv1review.StatementDisallowCommit))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, f.StatementDisallowLimit(mariadbv1review.StatementDisallowLimit))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, f.StatementDisallowOrderBy(mariadbv1review.StatementDisallowOrderBy))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, f.StatementDisallowTruncate(mariadbv1review.StatementDisallowTruncate))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, f.StatementInsertDisallowOrderByRand(mariadbv1review.StatementInsertDisallowOrderByRand))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, f.StatementInsertMustSpecifyColumn(mariadbv1review.StatementInsertMustSpecifyColumn))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME, f.StatementMaxExecutionTime(mariadbv1review.StatementMaxExecutionTime))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, f.StatementMergeAlterTable(mariadbv1review.StatementMergeAlterTable))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION, f.StatementRequireAlgorithmOption(mariadbv1review.StatementRequireAlgorithmOption))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION, f.StatementRequireLockOption(mariadbv1review.StatementRequireLockOption))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, f.StatementSelectNoSelectAll(mariadbv1review.StatementSelectNoSelectAll))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, f.StatementWhereNoLeadingWildcardLike(mariadbv1review.StatementWhereNoLeadingWildcardLike))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, f.StatementWhereRequireSelect(mariadbv1review.StatementWhereRequireSelect))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, f.StatementWhereRequireUpdateDelete(mariadbv1review.StatementWhereRequireUpdateDelete))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, f.SystemCharsetAllowlist(mariadbv1review.SystemCharsetAllowlist))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, f.SystemCollationAllowlist(mariadbv1review.SystemCollationAllowlist))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_TABLE_COMMENT, f.TableComment(mariadbv1review.TableComment))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, f.TableDisallowPartition(mariadbv1review.TableDisallowPartition))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, f.TableDropNaming(mariadbv1review.TableDropNaming))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, f.TableNoForeignKey(mariadbv1review.TableNoForeignKey))
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_TABLE_REQUIRE_PK, f.TableRequirePK(mariadbv1review.TableRequirePK))
}

func definesColumn(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		return n.Table != nil && slices.ContainsFunc(n.Columns, func(col *ast.ColumnDef) bool { return col != nil })
	case *ast.AlterTableStmt:
		if n.Table == nil {
			return false
		}
		return slices.ContainsFunc(n.Commands, func(cmd *ast.AlterTableCmd) bool {
			if cmd == nil {
				return false
			}
			switch cmd.Type {
			case ast.ATAddColumn:
				return cmd.Column != nil || len(cmd.Columns) > 0
			case ast.ATChangeColumn:
				return cmd.Column != nil
			default:
				return false
			}
		})
	default:
		return false
	}
}

func readsColumnType(node ast.Node) bool {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok || alter.Table == nil {
		return false
	}
	return slices.ContainsFunc(alter.Commands, func(cmd *ast.AlterTableCmd) bool {
		if cmd == nil || cmd.Column == nil || cmd.Column.TypeName == nil {
			return false
		}
		return cmd.Type == ast.ATChangeColumn || cmd.Type == ast.ATModifyColumn
	})
}

func dropsColumn(node ast.Node) bool {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok || alter.Table == nil {
		return false
	}
	return slices.ContainsFunc(alter.Commands, func(cmd *ast.AlterTableCmd) bool {
		return cmd != nil && cmd.Type == ast.ATDropColumn
	})
}

func isRenameIndex(node ast.Node) bool {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return false
	}
	return slices.ContainsFunc(alter.Commands, func(cmd *ast.AlterTableCmd) bool {
		return cmd != nil && cmd.Type == ast.ATRenameIndex
	})
}

func isDropDatabase(node ast.Node) bool {
	_, ok := node.(*ast.DropDatabaseStmt)
	return ok
}
