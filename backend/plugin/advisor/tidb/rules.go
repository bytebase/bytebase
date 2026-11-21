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
	extract := func(checkCtx advisor.Context) ([]advisor.V1Stmt[ast.Node], error) {
		stmts, err := getTiDBNodes(checkCtx)
		if err != nil {
			return nil, err
		}
		out := make([]advisor.V1Stmt[ast.Node], len(stmts))
		for i, s := range stmts {
			out[i] = advisor.V1Stmt[ast.Node]{Node: s.Node, Text: s.Text, BaseLine: s.BaseLine}
		}
		return out, nil
	}
	byteOffsetLine := func(sql string, offset int) int32 {
		return tidbparser.ByteOffsetToRunePosition(sql, offset).Line
	}
	e := advisor.RegisterMySQLFamily(storepb.Engine_TIDB, advisor.MySQLFamilyOptions[ast.Node]{
		Extract:             extract,
		ByteOffsetLine:      byteOffsetLine,
		TiDBQuirks:          true,
		DefinesColumn:       definesColumn,
		ReadsColumnType:     readsColumnType,
		DropsColumn:         dropsColumn,
		IsRenameIndex:       isRenameIndex,
		IsDropDatabase:      isDropDatabase,
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

	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, tidbPriorBackup(e))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, tidbDMLDryRun(e))
}

func tidbPriorBackup(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		if !checkCtx.EnablePriorBackup {
			return nil, nil
		}
		stmts, nodes, level, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		defaultDB := checkCtx.DBSchema.GetName()
		if defaultDB == "" {
			defaultDB = checkCtx.CurrentDatabase
		}
		facts := advisor.NewPriorBackupFacts(ctx, checkCtx, storepb.Engine_TIDB)
		findings := tidbv1review.PriorBackupCheck(nodes, defaultDB, e.SchemaOf(checkCtx), facts)
		return advisor.RenderPriorBackupFindings(title, level, findings, func(idx int) int {
			return Stmt{Node: stmts[idx].Node, Text: stmts[idx].Text, BaseLine: stmts[idx].BaseLine}.FirstTokenLine()
		}), nil
	}
}

func tidbDMLDryRun(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return e.DMLDryRun(advisor.V1DMLDryRunConfig[ast.Node]{
		Probes: tidbv1review.StatementDMLDryRunProbes,
		Judge:  tidbv1review.StatementDMLDryRun,
		Probe: func(ctx context.Context, prober db.Prober, stmt advisor.V1Stmt[ast.Node], _ []string) v1review.Probe {
			if reason, failed := probeTiDBDML(ctx, prober, Stmt{Node: stmt.Node, Text: stmt.Text, BaseLine: stmt.BaseLine}); failed {
				return v1review.Probe{Error: reason}
			}
			return v1review.Probe{}
		},
		FormatContent: func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("\"%s\" dry runs failed: %s", strings.TrimSpace(stmt.Text), f.Value)
		},
		Pos: func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) *storepb.Position {
			return e.PosFinding(stmt, v1review.Finding{})
		},
	})
}

func probeTiDBDML(ctx context.Context, prober db.Prober, ostmt Stmt) (string, bool) {
	explain := func(sql string) (string, bool) {
		if _, err := prober.Probe(ctx, db.ProbeContext{}, fmt.Sprintf("EXPLAIN %s", sql)); err != nil {
			return err.Error(), true
		}
		return "", false
	}
	node, ok := ostmt.Node.(*ast.BatchStmt)
	if !ok {
		return explain(strings.TrimRight(ostmt.TrimmedText(), ";"))
	}
	if loc, ok := dmlDryRunInnerLoc(node.DML); ok && loc.Start > 0 && loc.Start <= len(ostmt.Text) {
		dryRunSQL := ostmt.Text[:loc.Start] + "DRY RUN " + ostmt.Text[loc.Start:]
		if node.DryRun != ast.BatchDryRunNone {
			dryRunSQL = ostmt.Text
		}
		if _, err := prober.Probe(ctx, db.ProbeContext{Autocommit: true}, dryRunSQL); err != nil {
			return err.Error(), true
		}
		end := loc.End
		if end <= loc.Start || end > len(ostmt.Text) {
			end = len(ostmt.Text)
		}
		return explain(strings.TrimRight(strings.TrimSpace(ostmt.Text[loc.Start:end]), ";"))
	}
	return "failed to extract inner DML", true
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
		switch cmd.Type {
		case ast.ATAddColumn:
			return slices.ContainsFunc(addColumnTargets(cmd), func(col *ast.ColumnDef) bool { return col != nil })
		case ast.ATChangeColumn:
			return cmd.Column != nil
		default:
			return false
		}
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

func isRenameIndex(node ast.Node) bool {
	return hasAlterCmd(node, func(cmd *ast.AlterTableCmd) bool { return cmd.Type == ast.ATRenameIndex })
}

func isDropDatabase(node ast.Node) bool {
	_, ok := node.(*ast.DropDatabaseStmt)
	return ok
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
