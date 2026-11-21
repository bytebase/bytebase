// Package mysql implements the SQL advisor rules for MySQL.
package mysql

import (
	"slices"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

func init() {
	extract := func(checkCtx advisor.Context) ([]advisor.V1Stmt[ast.Node], error) {
		return advisor.ExtractNodes(checkCtx.ParsedStatements, mysqlparser.GetNode), nil
	}
	byteOffsetLine := func(sql string, offset int) int32 {
		return mysqlparser.ByteOffsetToRunePosition(sql, offset).Line
	}
	makePriorBackupStmt := func(node ast.Node, text string) mysqlv1review.PriorBackupStatement {
		return mysqlv1review.PriorBackupStatement{Node: node, Text: text}
	}
	f := advisor.NewMySQLFamily(advisor.MySQLFamilyOptions[ast.Node]{
		Extract:         extract,
		ByteOffsetLine:  byteOffsetLine,
		DefinesColumn:   definesColumn,
		ReadsColumnType: readsColumnType,
		DropsColumn:     dropsColumn,
		IsRenameIndex:   isRenameIndex,
		IsDropDatabase:  isDropDatabase,
		IsCreateTable:   isCreateTable,
	})

	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK, &advisor.BuiltinWalkThroughCheckAdvisor{})
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, advisor.NewMySQLPriorBackupAdvisor(storepb.Engine_MYSQL, extract, makePriorBackupStmt, mysqlv1review.PriorBackupCheck))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_ADVICE_ONLINE_MIGRATION, f.OnlineMigration(storepb.Engine_MYSQL, mysqlv1review.OnlineMigration))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION, f.MaxStatementsInTransaction())
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, advisor.NewMySQLRowLimitAdvisor(extract, byteOffsetLine, mysqlv1review.StatementInsertRowLimitProbes, mysqlv1review.StatementInsertRowLimit, code.InsertTooManyRows, "inserts", true))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, advisor.NewMySQLRowLimitAdvisor(extract, byteOffsetLine, mysqlv1review.StatementAffectedRowLimitProbes, mysqlv1review.StatementAffectedRowLimit, code.StatementAffectedRowExceedsLimit, "affected", false))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, advisor.NewMySQLDMLDryRunAdvisor(extract, byteOffsetLine, mysqlv1review.StatementDMLDryRunProbes, mysqlv1review.StatementDMLDryRun))

	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, f.ColumnAutoIncrementInitialValue(mysqlv1review.ColumnAutoIncrementInitialValue))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, f.ColumnAutoIncrementMustInteger(mysqlv1review.ColumnAutoIncrementMustInteger))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, f.ColumnAutoIncrementMustUnsigned(mysqlv1review.ColumnAutoIncrementMustUnsigned))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_COMMENT, f.ColumnComment(mysqlv1review.ColumnComment))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, f.ColumnCurrentTimeCountLimit(mysqlv1review.ColumnCurrentTimeCountLimit))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE, f.ColumnDisallowChange(mysqlv1review.ColumnDisallowChange))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, f.ColumnDisallowChangeType(mysqlv1review.ColumnDisallowChangeType))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, f.ColumnDisallowChangingOrder(mysqlv1review.ColumnDisallowChangingOrder))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, f.ColumnDisallowDropInIndex(mysqlv1review.ColumnDisallowDropInIndex))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, f.ColumnDisallowSetCharset(mysqlv1review.ColumnDisallowSetCharset))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, f.ColumnMaximumCharacterLength(mysqlv1review.ColumnMaximumCharacterLength))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, f.ColumnMaximumVarcharLength(mysqlv1review.ColumnMaximumVarcharLength))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_NO_NULL, f.ColumnNoNull(mysqlv1review.ColumnNoNull))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_REQUIRE_CHARSET, f.ColumnRequireCharset(mysqlv1review.ColumnRequireCharset))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_REQUIRE_COLLATION, f.ColumnRequireCollation(mysqlv1review.ColumnRequireCollation))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, f.ColumnRequireDefault(mysqlv1review.ColumnRequireDefault))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_REQUIRED, f.ColumnRequired(mysqlv1review.ColumnRequired))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, f.ColumnSetDefaultForNotNull(mysqlv1review.ColumnSetDefaultForNotNull))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, f.ColumnTypeDisallowList(mysqlv1review.ColumnTypeDisallowList))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, f.DatabaseDropEmptyDatabase(mysqlv1review.DatabaseDropEmptyDatabase))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB, f.EngineMySQLUseInnoDB(mysqlv1review.EngineMySQLUseInnoDB))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, f.IndexKeyNumberLimit(mysqlv1review.IndexKeyNumberLimit))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, f.IndexNoDuplicateColumn(mysqlv1review.IndexNoDuplicateColumn))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, f.IndexPKTypeLimit(mysqlv1review.IndexPKTypeLimit))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, f.IndexPrimaryKeyTypeAllowlist(mysqlv1review.IndexPrimaryKeyTypeAllowlist))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, f.IndexTotalNumberLimit(mysqlv1review.IndexTotalNumberLimit))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST, f.IndexTypeAllowList(mysqlv1review.IndexTypeAllowList))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, f.IndexTypeNoBlob(mysqlv1review.IndexTypeNoBlob))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, f.NamingAutoIncrementColumn(mysqlv1review.NamingAutoIncrementColumn))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_COLUMN, f.NamingColumn(mysqlv1review.NamingColumn))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_INDEX_FK, f.NamingForeignKey(mysqlv1review.NamingForeignKey))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, f.NamingIdentifierNoKeyword(mysqlv1review.NamingIdentifierNoKeyword))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_INDEX_IDX, f.NamingIndex(mysqlv1review.NamingIndex))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_TABLE, f.NamingTable(mysqlv1review.NamingTable))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_INDEX_UK, f.NamingUniqueKey(mysqlv1review.NamingUniqueKey))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, f.SchemaBackwardCompatibility(mysqlv1review.SchemaBackwardCompatibility))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, f.StatementDisallowCommit(mysqlv1review.StatementDisallowCommit))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, f.StatementDisallowLimit(mysqlv1review.StatementDisallowLimit))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, f.StatementDisallowOrderBy(mysqlv1review.StatementDisallowOrderBy))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, f.StatementDisallowTruncate(mysqlv1review.StatementDisallowTruncate))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, f.StatementInsertDisallowOrderByRand(mysqlv1review.StatementInsertDisallowOrderByRand))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, f.StatementInsertMustSpecifyColumn(mysqlv1review.StatementInsertMustSpecifyColumn))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME, f.StatementMaxExecutionTime(mysqlv1review.StatementMaxExecutionTime))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, f.StatementMergeAlterTable(mysqlv1review.StatementMergeAlterTable))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION, f.StatementRequireAlgorithmOption(mysqlv1review.StatementRequireAlgorithmOption))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION, f.StatementRequireLockOption(mysqlv1review.StatementRequireLockOption))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, f.StatementSelectNoSelectAll(mysqlv1review.StatementSelectNoSelectAll))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, f.StatementWhereDisallowFunctionsAndCalc(mysqlv1review.StatementWhereDisallowFunctionsAndCalculations))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT, f.StatementWhereMaximumLogicalOperatorCount(mysqlv1review.StatementWhereMaximumLogicalOperatorCount))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_NO_EQUAL_NULL, f.StatementWhereNoEqualNull(mysqlv1review.StatementWhereNoEqualNull))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, f.StatementWhereNoLeadingWildcardLike(mysqlv1review.StatementWhereNoLeadingWildcardLike))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, f.StatementWhereRequireSelect(mysqlv1review.StatementWhereRequireSelect))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, f.StatementWhereRequireUpdateDelete(mysqlv1review.StatementWhereRequireUpdateDelete))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, f.SystemCharsetAllowlist(mysqlv1review.SystemCharsetAllowlist))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, f.SystemCollationAllowlist(mysqlv1review.SystemCollationAllowlist))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_EVENT_DISALLOW_CREATE, f.SystemEventDisallowCreate(mysqlv1review.SystemEventDisallowCreate))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE, f.SystemFunctionDisallowCreate(mysqlv1review.SystemFunctionDisallowCreate))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST, f.SystemFunctionDisallowedList(mysqlv1review.SystemFunctionDisallowedList))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE, f.SystemProcedureDisallowCreate(mysqlv1review.SystemProcedureDisallowCreate))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_VIEW_DISALLOW_CREATE, f.SystemViewDisallowCreate(mysqlv1review.SystemViewDisallowCreate))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_COMMENT, f.TableComment(mysqlv1review.TableComment))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DDL, f.TableDisallowDDL(mysqlv1review.TableDisallowDDL))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DML, f.TableDisallowDML(mysqlv1review.TableDisallowDML))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, f.TableDisallowPartition(mysqlv1review.TableDisallowPartition))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_SET_CHARSET, f.TableDisallowSetCharset(mysqlv1review.TableDisallowSetCharset))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_TRIGGER, f.TableDisallowTrigger(mysqlv1review.TableDisallowTrigger))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, f.TableDropNaming(mysqlv1review.TableDropNaming))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_LIMIT_SIZE, f.TableLimitSize(mysqlv1review.TableLimitSize))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_NO_DUPLICATE_INDEX, f.TableNoDuplicateIndex(mysqlv1review.TableNoDuplicateIndex))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, f.TableNoForeignKey(mysqlv1review.TableNoForeignKey))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_REQUIRE_CHARSET, f.TableRequireCharset(mysqlv1review.TableRequireCharset))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_REQUIRE_COLLATION, f.TableRequireCollation(mysqlv1review.TableRequireCollation))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_REQUIRE_PK, f.TableRequirePK(mysqlv1review.TableRequirePK))
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_TEXT_FIELDS_TOTAL_LENGTH, f.TableTextFieldsTotalLength(mysqlv1review.TableTextFieldsTotalLength))
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

func isCreateTable(node ast.Node) bool {
	_, ok := node.(*ast.CreateTableStmt)
	return ok
}
