// Package mysql implements the SQL advisor rules for MySQL.
package mysql

import (
	"slices"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

func init() {
	e := advisor.RegisterMySQLFamily(storepb.Engine_MYSQL, advisor.MySQLFamilyOptions[ast.Node]{
		Extract:         advisor.NodeExtractor(mysqlparser.GetNode),
		DefinesColumn:   definesColumn,
		ReadsColumnType: readsColumnType,
		DropsColumn:     dropsColumn,
		IsCreateTable:   isCreateTable,
	}, advisor.MySQLFamilyRuleSet[ast.Node]{
		OnlineMigration:                           mysqlv1review.OnlineMigration,
		MaxStatementsInTransaction:                true,
		InsertRowLimitProbes:                      mysqlv1review.StatementInsertRowLimitProbes,
		InsertRowLimit:                            mysqlv1review.StatementInsertRowLimit,
		AffectedRowLimitProbes:                    mysqlv1review.StatementAffectedRowLimitProbes,
		AffectedRowLimit:                          mysqlv1review.StatementAffectedRowLimit,
		DMLDryRunProbes:                           mysqlv1review.StatementDMLDryRunProbes,
		DMLDryRun:                                 mysqlv1review.StatementDMLDryRun,
		ColumnAutoIncrementInitialValue:           mysqlv1review.ColumnAutoIncrementInitialValue,
		ColumnAutoIncrementMustInteger:            mysqlv1review.ColumnAutoIncrementMustInteger,
		ColumnAutoIncrementMustUnsigned:           mysqlv1review.ColumnAutoIncrementMustUnsigned,
		ColumnComment:                             mysqlv1review.ColumnComment,
		ColumnCurrentTimeCountLimit:               mysqlv1review.ColumnCurrentTimeCountLimit,
		ColumnDisallowChange:                      mysqlv1review.ColumnDisallowChange,
		ColumnDisallowChangeType:                  mysqlv1review.ColumnDisallowChangeType,
		ColumnDisallowChangingOrder:               mysqlv1review.ColumnDisallowChangingOrder,
		ColumnDisallowDropInIndex:                 mysqlv1review.ColumnDisallowDropInIndex,
		ColumnDisallowSetCharset:                  mysqlv1review.ColumnDisallowSetCharset,
		ColumnMaximumCharacterLength:              mysqlv1review.ColumnMaximumCharacterLength,
		ColumnMaximumVarcharLength:                mysqlv1review.ColumnMaximumVarcharLength,
		ColumnNoNull:                              mysqlv1review.ColumnNoNull,
		ColumnRequireCharset:                      mysqlv1review.ColumnRequireCharset,
		ColumnRequireCollation:                    mysqlv1review.ColumnRequireCollation,
		ColumnRequireDefault:                      mysqlv1review.ColumnRequireDefault,
		ColumnRequired:                            mysqlv1review.ColumnRequired,
		ColumnSetDefaultForNotNull:                mysqlv1review.ColumnSetDefaultForNotNull,
		ColumnTypeDisallowList:                    mysqlv1review.ColumnTypeDisallowList,
		DatabaseDropEmptyDatabase:                 mysqlv1review.DatabaseDropEmptyDatabase,
		EngineMySQLUseInnoDB:                      mysqlv1review.EngineMySQLUseInnoDB,
		IndexKeyNumberLimit:                       mysqlv1review.IndexKeyNumberLimit,
		IndexNoDuplicateColumn:                    mysqlv1review.IndexNoDuplicateColumn,
		IndexPKTypeLimit:                          mysqlv1review.IndexPKTypeLimit,
		IndexPrimaryKeyTypeAllowlist:              mysqlv1review.IndexPrimaryKeyTypeAllowlist,
		IndexTotalNumberLimit:                     mysqlv1review.IndexTotalNumberLimit,
		IndexTypeAllowList:                        mysqlv1review.IndexTypeAllowList,
		IndexTypeNoBlob:                           mysqlv1review.IndexTypeNoBlob,
		NamingAutoIncrementColumn:                 mysqlv1review.NamingAutoIncrementColumn,
		NamingColumn:                              mysqlv1review.NamingColumn,
		NamingForeignKey:                          mysqlv1review.NamingForeignKey,
		NamingIdentifierNoKeyword:                 mysqlv1review.NamingIdentifierNoKeyword,
		NamingIndex:                               mysqlv1review.NamingIndex,
		NamingTable:                               mysqlv1review.NamingTable,
		NamingUniqueKey:                           mysqlv1review.NamingUniqueKey,
		SchemaBackwardCompatibility:               mysqlv1review.SchemaBackwardCompatibility,
		StatementDisallowCommit:                   mysqlv1review.StatementDisallowCommit,
		StatementDisallowLimit:                    mysqlv1review.StatementDisallowLimit,
		StatementDisallowOrderBy:                  mysqlv1review.StatementDisallowOrderBy,
		StatementDisallowTruncate:                 mysqlv1review.StatementDisallowTruncate,
		StatementInsertDisallowOrderByRand:        mysqlv1review.StatementInsertDisallowOrderByRand,
		StatementInsertMustSpecifyColumn:          mysqlv1review.StatementInsertMustSpecifyColumn,
		StatementMaxExecutionTime:                 mysqlv1review.StatementMaxExecutionTime,
		StatementMergeAlterTable:                  mysqlv1review.StatementMergeAlterTable,
		StatementRequireAlgorithmOption:           mysqlv1review.StatementRequireAlgorithmOption,
		StatementRequireLockOption:                mysqlv1review.StatementRequireLockOption,
		StatementSelectNoSelectAll:                mysqlv1review.StatementSelectNoSelectAll,
		StatementWhereDisallowFunctionsAndCalc:    mysqlv1review.StatementWhereDisallowFunctionsAndCalculations,
		StatementWhereMaximumLogicalOperatorCount: mysqlv1review.StatementWhereMaximumLogicalOperatorCount,
		StatementWhereNoEqualNull:                 mysqlv1review.StatementWhereNoEqualNull,
		StatementWhereNoLeadingWildcardLike:       mysqlv1review.StatementWhereNoLeadingWildcardLike,
		StatementWhereRequireSelect:               mysqlv1review.StatementWhereRequireSelect,
		StatementWhereRequireUpdateDelete:         mysqlv1review.StatementWhereRequireUpdateDelete,
		SystemCharsetAllowlist:                    mysqlv1review.SystemCharsetAllowlist,
		SystemCollationAllowlist:                  mysqlv1review.SystemCollationAllowlist,
		SystemEventDisallowCreate:                 mysqlv1review.SystemEventDisallowCreate,
		SystemFunctionDisallowCreate:              mysqlv1review.SystemFunctionDisallowCreate,
		SystemFunctionDisallowedList:              mysqlv1review.SystemFunctionDisallowedList,
		SystemProcedureDisallowCreate:             mysqlv1review.SystemProcedureDisallowCreate,
		SystemViewDisallowCreate:                  mysqlv1review.SystemViewDisallowCreate,
		TableComment:                              mysqlv1review.TableComment,
		TableDisallowDDL:                          mysqlv1review.TableDisallowDDL,
		TableDisallowDML:                          mysqlv1review.TableDisallowDML,
		TableDisallowPartition:                    mysqlv1review.TableDisallowPartition,
		TableDisallowSetCharset:                   mysqlv1review.TableDisallowSetCharset,
		TableDisallowTrigger:                      mysqlv1review.TableDisallowTrigger,
		TableDropNaming:                           mysqlv1review.TableDropNaming,
		TableLimitSize:                            mysqlv1review.TableLimitSize,
		TableNoDuplicateIndex:                     mysqlv1review.TableNoDuplicateIndex,
		TableNoForeignKey:                         mysqlv1review.TableNoForeignKey,
		TableRequireCharset:                       mysqlv1review.TableRequireCharset,
		TableRequireCollation:                     mysqlv1review.TableRequireCollation,
		TableRequirePK:                            mysqlv1review.TableRequirePK,
		TableTextFieldsTotalLength:                mysqlv1review.TableTextFieldsTotalLength,
	})

	advisor.RegisterRules(storepb.Engine_MYSQL, map[storepb.SQLReviewRule_Type]advisor.Advisor{
		storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK: advisor.NewMySQLPriorBackupAdvisor(e, storepb.Engine_MYSQL, func(node ast.Node, text string) mysqlv1review.PriorBackupStatement {
			return mysqlv1review.PriorBackupStatement{Node: node, Text: text}
		}, mysqlv1review.PriorBackupCheck),
	})
}

func hasAlterCmd(node ast.Node, match func(*ast.AlterTableCmd) bool) bool {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok || alter.Table == nil {
		return false
	}
	return slices.ContainsFunc(alter.Commands, func(cmd *ast.AlterTableCmd) bool {
		return cmd != nil && match(cmd)
	})
}

func definesColumn(node ast.Node) bool {
	if n, ok := node.(*ast.CreateTableStmt); ok {
		return n.Table != nil && slices.ContainsFunc(n.Columns, func(col *ast.ColumnDef) bool { return col != nil })
	}
	return hasAlterCmd(node, func(cmd *ast.AlterTableCmd) bool {
		return (cmd.Type == ast.ATAddColumn && (cmd.Column != nil || len(cmd.Columns) > 0)) ||
			(cmd.Type == ast.ATChangeColumn && cmd.Column != nil)
	})
}

func readsColumnType(node ast.Node) bool {
	return hasAlterCmd(node, func(cmd *ast.AlterTableCmd) bool {
		return cmd.Column != nil && cmd.Column.TypeName != nil && (cmd.Type == ast.ATChangeColumn || cmd.Type == ast.ATModifyColumn)
	})
}

func dropsColumn(node ast.Node) bool {
	return hasAlterCmd(node, func(cmd *ast.AlterTableCmd) bool { return cmd.Type == ast.ATDropColumn })
}

func isCreateTable(node ast.Node) bool {
	_, ok := node.(*ast.CreateTableStmt)
	return ok
}
