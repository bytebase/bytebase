// Package mariadb implements the SQL advisor rules for MariaDB.
package mariadb

import (
	"slices"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

func init() {
	extract := advisor.NodeExtractor(mariadbparser.GetNode)
	byteOffsetLine := func(sql string, offset int) int32 {
		return mariadbparser.ByteOffsetToRunePosition(sql, offset).Line
	}
	makePriorBackupStmt := func(node ast.Node, text string) mariadbv1review.PriorBackupStatement {
		return mariadbv1review.PriorBackupStatement{Node: node, Text: text}
	}

	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK,
		advisor.NewMySQLPriorBackupAdvisor(storepb.Engine_MARIADB, extract, makePriorBackupStmt, mariadbv1review.PriorBackupCheck))

	advisor.RegisterMySQLFamily(storepb.Engine_MARIADB, advisor.MySQLFamilyOptions[ast.Node]{
		Extract:         extract,
		ByteOffsetLine:  byteOffsetLine,
		DefinesColumn:   definesColumn,
		ReadsColumnType: readsColumnType,
		DropsColumn:     dropsColumn,
		IsRenameIndex:   isRenameIndex,
		IsDropDatabase:  isDropDatabase,
	}, advisor.MySQLFamilyRuleSet[ast.Node]{
		OnlineMigration:                     mariadbv1review.OnlineMigration,
		InsertRowLimitProbes:                mariadbv1review.StatementInsertRowLimitProbes,
		InsertRowLimit:                      mariadbv1review.StatementInsertRowLimit,
		AffectedRowLimitProbes:              mariadbv1review.StatementAffectedRowLimitProbes,
		AffectedRowLimit:                    mariadbv1review.StatementAffectedRowLimit,
		DMLDryRunProbes:                     mariadbv1review.StatementDMLDryRunProbes,
		DMLDryRun:                           mariadbv1review.StatementDMLDryRun,
		ColumnAutoIncrementInitialValue:     mariadbv1review.ColumnAutoIncrementInitialValue,
		ColumnAutoIncrementMustInteger:      mariadbv1review.ColumnAutoIncrementMustInteger,
		ColumnAutoIncrementMustUnsigned:     mariadbv1review.ColumnAutoIncrementMustUnsigned,
		ColumnComment:                       mariadbv1review.ColumnComment,
		ColumnCurrentTimeCountLimit:         mariadbv1review.ColumnCurrentTimeCountLimit,
		ColumnDisallowChange:                mariadbv1review.ColumnDisallowChange,
		ColumnDisallowChangeType:            mariadbv1review.ColumnDisallowChangeType,
		ColumnDisallowChangingOrder:         mariadbv1review.ColumnDisallowChangingOrder,
		ColumnDisallowDropInIndex:           mariadbv1review.ColumnDisallowDropInIndex,
		ColumnDisallowSetCharset:            mariadbv1review.ColumnDisallowSetCharset,
		ColumnMaximumCharacterLength:        mariadbv1review.ColumnMaximumCharacterLength,
		ColumnMaximumVarcharLength:          mariadbv1review.ColumnMaximumVarcharLength,
		ColumnNoNull:                        mariadbv1review.ColumnNoNull,
		ColumnRequireDefault:                mariadbv1review.ColumnRequireDefault,
		ColumnRequired:                      mariadbv1review.ColumnRequired,
		ColumnSetDefaultForNotNull:          mariadbv1review.ColumnSetDefaultForNotNull,
		ColumnTypeDisallowList:              mariadbv1review.ColumnTypeDisallowList,
		DatabaseDropEmptyDatabase:           mariadbv1review.DatabaseDropEmptyDatabase,
		EngineMySQLUseInnoDB:                mariadbv1review.EngineMySQLUseInnoDB,
		IndexKeyNumberLimit:                 mariadbv1review.IndexKeyNumberLimit,
		IndexNoDuplicateColumn:              mariadbv1review.IndexNoDuplicateColumn,
		IndexPKTypeLimit:                    mariadbv1review.IndexPKTypeLimit,
		IndexTotalNumberLimit:               mariadbv1review.IndexTotalNumberLimit,
		IndexTypeNoBlob:                     mariadbv1review.IndexTypeNoBlob,
		NamingAutoIncrementColumn:           mariadbv1review.NamingAutoIncrementColumn,
		NamingColumn:                        mariadbv1review.NamingColumn,
		NamingForeignKey:                    mariadbv1review.NamingForeignKey,
		NamingIndex:                         mariadbv1review.NamingIndex,
		NamingTable:                         mariadbv1review.NamingTable,
		NamingUniqueKey:                     mariadbv1review.NamingUniqueKey,
		SchemaBackwardCompatibility:         mariadbv1review.SchemaBackwardCompatibility,
		StatementDisallowCommit:             mariadbv1review.StatementDisallowCommit,
		StatementDisallowLimit:              mariadbv1review.StatementDisallowLimit,
		StatementDisallowOrderBy:            mariadbv1review.StatementDisallowOrderBy,
		StatementDisallowTruncate:           mariadbv1review.StatementDisallowTruncate,
		StatementInsertDisallowOrderByRand:  mariadbv1review.StatementInsertDisallowOrderByRand,
		StatementInsertMustSpecifyColumn:    mariadbv1review.StatementInsertMustSpecifyColumn,
		StatementMaxExecutionTime:           mariadbv1review.StatementMaxExecutionTime,
		StatementMergeAlterTable:            mariadbv1review.StatementMergeAlterTable,
		StatementRequireAlgorithmOption:     mariadbv1review.StatementRequireAlgorithmOption,
		StatementRequireLockOption:          mariadbv1review.StatementRequireLockOption,
		StatementSelectNoSelectAll:          mariadbv1review.StatementSelectNoSelectAll,
		StatementWhereNoLeadingWildcardLike: mariadbv1review.StatementWhereNoLeadingWildcardLike,
		StatementWhereRequireSelect:         mariadbv1review.StatementWhereRequireSelect,
		StatementWhereRequireUpdateDelete:   mariadbv1review.StatementWhereRequireUpdateDelete,
		SystemCharsetAllowlist:              mariadbv1review.SystemCharsetAllowlist,
		SystemCollationAllowlist:            mariadbv1review.SystemCollationAllowlist,
		TableComment:                        mariadbv1review.TableComment,
		TableDisallowPartition:              mariadbv1review.TableDisallowPartition,
		TableDropNaming:                     mariadbv1review.TableDropNaming,
		TableNoForeignKey:                   mariadbv1review.TableNoForeignKey,
		TableRequirePK:                      mariadbv1review.TableRequirePK,
	})
}

func hasAlterCmd(node ast.Node, requireTable bool, match func(*ast.AlterTableCmd) bool) bool {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok || (requireTable && alter.Table == nil) {
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
	return hasAlterCmd(node, true, func(cmd *ast.AlterTableCmd) bool {
		switch cmd.Type {
		case ast.ATAddColumn:
			return cmd.Column != nil || len(cmd.Columns) > 0
		case ast.ATChangeColumn:
			return cmd.Column != nil
		default:
			return false
		}
	})
}

func readsColumnType(node ast.Node) bool {
	return hasAlterCmd(node, true, func(cmd *ast.AlterTableCmd) bool {
		return cmd.Column != nil && cmd.Column.TypeName != nil && (cmd.Type == ast.ATChangeColumn || cmd.Type == ast.ATModifyColumn)
	})
}

func dropsColumn(node ast.Node) bool {
	return hasAlterCmd(node, true, func(cmd *ast.AlterTableCmd) bool { return cmd.Type == ast.ATDropColumn })
}

func isRenameIndex(node ast.Node) bool {
	return hasAlterCmd(node, false, func(cmd *ast.AlterTableCmd) bool { return cmd.Type == ast.ATRenameIndex })
}

func isDropDatabase(node ast.Node) bool {
	_, ok := node.(*ast.DropDatabaseStmt)
	return ok
}
