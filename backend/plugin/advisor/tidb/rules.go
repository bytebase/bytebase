package tidb

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

func init() {
	e := advisor.RegisterMySQLFamily(storepb.Engine_TIDB, advisor.MySQLFamilyOptions[ast.Node]{
		Extract:             advisor.NodeExtractor(tidbparser.GetNode),
		TiDBQuirks:          true,
		DefinesColumn:       definesColumn,
		ReadsColumnType:     readsColumnType,
		DropsColumn:         dropsColumn,
		IsAlterTable:        isAlterTable,
		CreateTableLocStart: createTableLocStart,
	}, advisor.MySQLFamilyRuleSet[ast.Node]{
		ColumnAutoIncrementInitialValue:     tidbv1review.ColumnAutoIncrementInitialValue,
		ColumnAutoIncrementMustInteger:      tidbv1review.ColumnAutoIncrementMustInteger,
		ColumnAutoIncrementMustUnsigned:     tidbv1review.ColumnAutoIncrementMustUnsigned,
		ColumnComment:                       tidbv1review.ColumnComment,
		ColumnCurrentTimeCountLimit:         tidbv1review.ColumnCurrentTimeCountLimit,
		ColumnDisallowChange:                tidbv1review.ColumnDisallowChange,
		ColumnDisallowChangeType:            tidbv1review.ColumnDisallowChangeType,
		ColumnDisallowChangingOrder:         tidbv1review.ColumnDisallowChangingOrder,
		ColumnDisallowDropInIndex:           tidbv1review.ColumnDisallowDropInIndex,
		ColumnDisallowSetCharset:            tidbv1review.ColumnDisallowSetCharset,
		ColumnMaximumCharacterLength:        tidbv1review.ColumnMaximumCharacterLength,
		ColumnNoNull:                        tidbv1review.ColumnNoNull,
		ColumnRequireDefault:                tidbv1review.ColumnRequireDefault,
		ColumnRequired:                      tidbv1review.ColumnRequired,
		ColumnSetDefaultForNotNull:          tidbv1review.ColumnSetDefaultForNotNull,
		ColumnTypeDisallowList:              tidbv1review.ColumnTypeDisallowList,
		DatabaseDropEmptyDatabase:           tidbv1review.DatabaseDropEmptyDatabase,
		IndexKeyNumberLimit:                 tidbv1review.IndexKeyNumberLimit,
		IndexNoDuplicateColumn:              tidbv1review.IndexNoDuplicateColumn,
		IndexPKTypeLimit:                    tidbv1review.IndexPKTypeLimit,
		IndexPrimaryKeyTypeAllowlist:        tidbv1review.IndexPrimaryKeyTypeAllowlist,
		IndexTotalNumberLimit:               tidbv1review.IndexTotalNumberLimit,
		IndexTypeNoBlob:                     tidbv1review.IndexTypeNoBlob,
		NamingAutoIncrementColumn:           tidbv1review.NamingAutoIncrementColumn,
		NamingColumn:                        tidbv1review.NamingColumn,
		NamingForeignKey:                    tidbv1review.NamingForeignKey,
		NamingIndex:                         tidbv1review.NamingIndex,
		NamingTable:                         tidbv1review.NamingTable,
		NamingUniqueKey:                     tidbv1review.NamingUniqueKey,
		SchemaBackwardCompatibility:         tidbv1review.SchemaBackwardCompatibility,
		StatementDisallowCommit:             tidbv1review.StatementDisallowCommit,
		StatementDisallowLimit:              tidbv1review.StatementDisallowLimit,
		StatementDisallowOrderBy:            tidbv1review.StatementDisallowOrderBy,
		StatementInsertDisallowOrderByRand:  tidbv1review.StatementInsertDisallowOrderByRand,
		StatementInsertMustSpecifyColumn:    tidbv1review.StatementInsertMustSpecifyColumn,
		StatementMergeAlterTable:            tidbv1review.StatementMergeAlterTable,
		StatementSelectNoSelectAll:          tidbv1review.StatementSelectNoSelectAll,
		StatementWhereNoLeadingWildcardLike: tidbv1review.StatementWhereNoLeadingWildcardLike,
		StatementWhereRequireSelect:         tidbv1review.StatementWhereRequireSelect,
		StatementWhereRequireUpdateDelete:   tidbv1review.StatementWhereRequireUpdateDelete,
		SystemCharsetAllowlist:              tidbv1review.SystemCharsetAllowlist,
		SystemCollationAllowlist:            tidbv1review.SystemCollationAllowlist,
		TableComment:                        tidbv1review.TableComment,
		TableDisallowPartition:              tidbv1review.TableDisallowPartition,
		TableDropNaming:                     tidbv1review.TableDropNaming,
		TableNoForeignKey:                   tidbv1review.TableNoForeignKey,
		TableRequirePK:                      tidbv1review.TableRequirePK,
	})

	advisor.RegisterRules(storepb.Engine_TIDB, map[storepb.SQLReviewRule_Type]advisor.Advisor{
		storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK: e.PriorBackup(advisor.V1PriorBackupConfig[ast.Node]{
			Engine: storepb.Engine_TIDB,
			Check: func(checkCtx advisor.Context, _ []advisor.V1Stmt[ast.Node], nodes []ast.Node, facts v1review.PriorBackupFacts) ([]v1review.Finding, error) {
				defaultDB := checkCtx.DBSchema.GetName()
				if defaultDB == "" {
					defaultDB = checkCtx.CurrentDatabase
				}
				return tidbv1review.PriorBackupCheck(nodes, defaultDB, e.SchemaOf(checkCtx), facts), nil
			},
			MixedLine: e.FirstTokenLine,
		}),
		storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN: e.DMLDryRun(advisor.V1DMLDryRunConfig[ast.Node]{
			Probes:        tidbv1review.StatementDMLDryRunProbes,
			Judge:         tidbv1review.StatementDMLDryRun,
			StatementText: func(stmt advisor.V1Stmt[ast.Node]) string { return strings.TrimSpace(stmt.Text) },
			Probe: func(ctx context.Context, prober db.Prober, stmt advisor.V1Stmt[ast.Node], _ []string) v1review.Probe {
				return probeTiDBDML(ctx, prober, e, stmt)
			},
			Pos: e.PosStatement,
		}),
	})
}

func probeTiDBDML(ctx context.Context, prober db.Prober, e *advisor.V1Engine[ast.Node], stmt advisor.V1Stmt[ast.Node]) v1review.Probe {
	explain := func(sql string) v1review.Probe {
		return advisor.ExplainProbe(ctx, prober, db.ProbeContext{}, fmt.Sprintf("EXPLAIN %s", sql))
	}
	node, ok := stmt.Node.(*ast.BatchStmt)
	if !ok {
		return explain(e.TrimmedText(stmt.Text))
	}
	if loc, ok := dmlDryRunInnerLoc(node.DML); ok && loc.Start > 0 && loc.Start <= len(stmt.Text) {
		dryRunSQL := stmt.Text[:loc.Start] + "DRY RUN " + stmt.Text[loc.Start:]
		if node.DryRun != ast.BatchDryRunNone {
			dryRunSQL = stmt.Text
		}
		if p := advisor.ExplainProbe(ctx, prober, db.ProbeContext{Autocommit: true}, dryRunSQL); p.Error != "" {
			return p
		}
		end := loc.End
		if end <= loc.Start || end > len(stmt.Text) {
			end = len(stmt.Text)
		}
		return explain(e.TrimmedText(stmt.Text[loc.Start:end]))
	}
	return v1review.Probe{Error: "failed to extract inner DML"}
}

func dmlDryRunInnerLoc(n ast.StmtNode) (ast.Loc, bool) {
	switch s := n.(type) {
	case *ast.DeleteStmt:
		return s.Loc, true
	case *ast.UpdateStmt:
		return s.Loc, true
	case *ast.InsertStmt:
		return s.Loc, true
	default:
		return ast.Loc{}, false
	}
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
		return (cmd.Type == ast.ATAddColumn && (cmd.Column != nil || slices.ContainsFunc(cmd.Columns, func(col *ast.ColumnDef) bool { return col != nil }))) ||
			(cmd.Type == ast.ATChangeColumn && cmd.Column != nil)
	})
}

func readsColumnType(node ast.Node) bool {
	return hasAlterCmd(node, func(cmd *ast.AlterTableCmd) bool {
		if cmd.Column == nil || cmd.Column.TypeName == nil {
			return false
		}
		switch cmd.Type {
		case ast.ATChangeColumn:
			return cmd.Name != ""
		case ast.ATModifyColumn:
			return cmd.Column.Name != ""
		default:
			return false
		}
	})
}

func dropsColumn(node ast.Node) bool {
	return hasAlterCmd(node, func(cmd *ast.AlterTableCmd) bool { return cmd.Type == ast.ATDropColumn })
}

func isAlterTable(node ast.Node) bool {
	_, ok := node.(*ast.AlterTableStmt)
	return ok
}

func createTableLocStart(node ast.Node) (int, bool) {
	if n, ok := node.(*ast.CreateTableStmt); ok {
		return n.Loc.Start, true
	}
	return 0, false
}
