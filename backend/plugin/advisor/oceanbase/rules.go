// Package oceanbase implements the SQL advisor rules for OceanBase.
package oceanbase

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

func init() {
	extract := advisor.NodeExtractor(oceanbaseparser.GetNode)
	byteOffsetLine := func(sql string, offset int) int32 {
		return oceanbaseparser.ByteOffsetToRunePosition(sql, offset).Line
	}
	e := advisor.RegisterMySQLFamily(storepb.Engine_OCEANBASE, advisor.MySQLFamilyOptions[ast.Node]{
		Extract:         extract,
		ByteOffsetLine:  byteOffsetLine,
		DefinesColumn:   definesColumn,
		ReadsColumnType: readsColumnType,
		DropsColumn:     dropsColumn,
		IsRenameIndex:   isRenameIndex,
		IsDropDatabase:  isDropDatabase,
	}, advisor.MySQLFamilyRuleSet[ast.Node]{
		DMLDryRunProbes:                     oceanbasev1review.StatementDMLDryRunProbes,
		DMLDryRun:                           oceanbasev1review.StatementDMLDryRun,
		ColumnAutoIncrementInitialValue:     oceanbasev1review.ColumnAutoIncrementInitialValue,
		ColumnAutoIncrementMustInteger:      oceanbasev1review.ColumnAutoIncrementMustInteger,
		ColumnAutoIncrementMustUnsigned:     oceanbasev1review.ColumnAutoIncrementMustUnsigned,
		ColumnComment:                       oceanbasev1review.ColumnComment,
		ColumnCurrentTimeCountLimit:         oceanbasev1review.ColumnCurrentTimeCountLimit,
		ColumnDisallowChange:                oceanbasev1review.ColumnDisallowChange,
		ColumnDisallowChangeType:            oceanbasev1review.ColumnDisallowChangeType,
		ColumnDisallowChangingOrder:         oceanbasev1review.ColumnDisallowChangingOrder,
		ColumnDisallowDrop:                  oceanbasev1review.ColumnDisallowDrop,
		ColumnDisallowDropInIndex:           oceanbasev1review.ColumnDisallowDropInIndex,
		ColumnDisallowSetCharset:            oceanbasev1review.ColumnDisallowSetCharset,
		ColumnMaximumCharacterLength:        oceanbasev1review.ColumnMaximumCharacterLength,
		ColumnMaximumVarcharLength:          oceanbasev1review.ColumnMaximumVarcharLength,
		ColumnNoNull:                        oceanbasev1review.ColumnNoNull,
		ColumnRequireDefault:                oceanbasev1review.ColumnRequireDefault,
		ColumnRequired:                      oceanbasev1review.ColumnRequired,
		ColumnSetDefaultForNotNull:          oceanbasev1review.ColumnSetDefaultForNotNull,
		ColumnTypeDisallowList:              oceanbasev1review.ColumnTypeDisallowList,
		DatabaseDropEmptyDatabase:           oceanbasev1review.DatabaseDropEmptyDatabase,
		IndexKeyNumberLimit:                 oceanbasev1review.IndexKeyNumberLimit,
		IndexNoDuplicateColumn:              oceanbasev1review.IndexNoDuplicateColumn,
		IndexPKTypeLimit:                    oceanbasev1review.IndexPKTypeLimit,
		IndexPrimaryKeyTypeAllowlist:        oceanbasev1review.IndexPrimaryKeyTypeAllowlist,
		IndexTotalNumberLimit:               oceanbasev1review.IndexTotalNumberLimit,
		IndexTypeNoBlob:                     oceanbasev1review.IndexTypeNoBlob,
		NamingAutoIncrementColumn:           oceanbasev1review.NamingAutoIncrementColumn,
		NamingColumn:                        oceanbasev1review.NamingColumn,
		NamingForeignKey:                    oceanbasev1review.NamingForeignKey,
		NamingIndex:                         oceanbasev1review.NamingIndex,
		NamingTable:                         oceanbasev1review.NamingTable,
		NamingUniqueKey:                     oceanbasev1review.NamingUniqueKey,
		SchemaBackwardCompatibility:         oceanbasev1review.SchemaBackwardCompatibility,
		StatementAddColumnWithoutPosition:   oceanbasev1review.StatementAddColumnWithoutPosition,
		StatementDisallowCommit:             oceanbasev1review.StatementDisallowCommit,
		StatementDisallowLimit:              oceanbasev1review.StatementDisallowLimit,
		StatementDisallowOrderBy:            oceanbasev1review.StatementDisallowOrderBy,
		StatementDisallowTruncate:           oceanbasev1review.StatementDisallowTruncate,
		StatementInsertDisallowOrderByRand:  oceanbasev1review.StatementInsertDisallowOrderByRand,
		StatementInsertMustSpecifyColumn:    oceanbasev1review.StatementInsertMustSpecifyColumn,
		StatementMergeAlterTable:            oceanbasev1review.StatementMergeAlterTable,
		StatementSelectNoSelectAll:          oceanbasev1review.StatementSelectNoSelectAll,
		StatementWhereNoLeadingWildcardLike: oceanbasev1review.StatementWhereNoLeadingWildcardLike,
		StatementWhereRequireSelect:         oceanbasev1review.StatementWhereRequireSelect,
		StatementWhereRequireUpdateDelete:   oceanbasev1review.StatementWhereRequireUpdateDelete,
		SystemCharsetAllowlist:              oceanbasev1review.SystemCharsetAllowlist,
		SystemCollationAllowlist:            oceanbasev1review.SystemCollationAllowlist,
		TableComment:                        oceanbasev1review.TableComment,
		TableDisallowPartition:              oceanbasev1review.TableDisallowPartition,
		TableDropNaming:                     oceanbasev1review.TableDropNaming,
		TableNoForeignKey:                   oceanbasev1review.TableNoForeignKey,
		TableRequirePK:                      oceanbasev1review.TableRequirePK,
	})

	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
		oceanbaseRowLimitAdvisor(e, oceanbasev1review.StatementInsertRowLimitProbes, oceanbasev1review.StatementInsertRowLimit, code.InsertTooManyRows, "inserts"))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
		oceanbaseRowLimitAdvisor(e, oceanbasev1review.StatementAffectedRowLimitProbes, oceanbasev1review.StatementAffectedRowLimit, code.StatementAffectedRowExceedsLimit, "affected"))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_OFFLINE_DDL, disallowOfflineDDL(e))
}

func disallowOfflineDDL(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		lookup, err := advisor.ColumnExtraOf(checkCtx)
		if err != nil {
			return nil, err
		}
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			stmtNodes := nodes[i : i+1 : i+1]
			extras := make(map[v1review.ColumnRef]string)
			for _, ref := range oceanbasev1review.StatementDisallowOfflineDDLLookups(stmtNodes, checkCtx.CurrentDatabase) {
				if extra, err := lookup.ColumnExtra(ctx, ref.Database, ref.Table, ref.Column); err == nil {
					extras[ref] = extra
				}
			}
			for _, f := range oceanbasev1review.StatementDisallowOfflineDDL(stmtNodes, checkCtx.CurrentDatabase, extras) {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.StatementOfflineDDL.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("%s is an offline DDL operation.", f.Name),
					StartPosition: e.PosFindingOrFirstLine(stmt, f),
				})
			}
		}
		return adviceList, nil
	}
}

func oceanbaseRowLimitAdvisor(
	e *advisor.V1Engine[ast.Node],
	probesFn func([]ast.Node) []int,
	judgeFn func([]ast.Node, map[int]v1review.Probe, int) []v1review.Finding,
	exceededCode code.Code,
	verb string,
) advisor.Func {
	return e.RowLimit(advisor.V1RowLimitConfig[ast.Node]{
		Probes:       probesFn,
		Judge:        judgeFn,
		ExceededCode: exceededCode,
		ExceededVerb: verb,
		StatementText: func(stmt advisor.V1Stmt[ast.Node], _ ast.Node) string {
			return stmt.Text
		},
		Estimate: func(ctx context.Context, checkCtx advisor.Context, stmt advisor.V1Stmt[ast.Node], _ ast.Node, _ []string) (v1review.Probe, error) {
			return explainRows(ctx, checkCtx, stmt.Text)
		},
		BudgetPos: func(stmt advisor.V1Stmt[ast.Node], node ast.Node) *storepb.Position {
			return nodePosition(e, stmt, node)
		},
		Pos: e.PosFindingOrFirstLine,
	})
}

func explainRows(ctx context.Context, checkCtx advisor.Context, text string) (v1review.Probe, error) {
	prober, err := advisor.ProbeFor(checkCtx)
	if err != nil {
		return v1review.Probe{}, err
	}
	res, err := prober.Probe(ctx, db.ProbeContext{}, fmt.Sprintf("EXPLAIN format=json %s", text))
	if err != nil {
		//nolint:nilerr // a refused statement is the probe finding, reported as advice.
		return v1review.Probe{Error: err.Error()}, nil
	}
	rows, err := estimatedRowsFromPlan(res)
	if err != nil {
		//nolint:nilerr // an unreadable plan is the probe finding, reported as advice.
		return v1review.Probe{CountError: err.Error()}, nil
	}
	return v1review.Probe{Rows: rows}, nil
}

func nodePosition(e *advisor.V1Engine[ast.Node], stmt advisor.V1Stmt[ast.Node], node ast.Node) *storepb.Position {
	var loc ast.Loc
	switch n := node.(type) {
	case *ast.InsertStmt:
		loc = n.Loc
	case *ast.UpdateStmt:
		loc = n.Loc
	case *ast.DeleteStmt:
		loc = n.Loc
	default:
	}
	if loc.Start < 0 {
		loc.Start = 0
	}
	return e.PosFindingOrFirstLine(stmt, v1review.Finding{Start: loc.Start, End: loc.Start})
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
