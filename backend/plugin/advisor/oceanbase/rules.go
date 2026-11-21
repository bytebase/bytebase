// Package oceanbase implements the SQL advisor rules for OceanBase.
package oceanbase

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

func init() {
	extract := func(checkCtx advisor.Context) ([]advisor.V1Stmt[ast.Node], error) {
		return advisor.ExtractNodes(checkCtx.ParsedStatements, oceanbaseparser.GetNode), nil
	}
	byteOffsetLine := func(sql string, offset int) int32 {
		return oceanbaseparser.ByteOffsetToRunePosition(sql, offset).Line
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

	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK, &advisor.BuiltinWalkThroughCheckAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, oceanbaseRowLimitAdvisor(oceanbasev1review.StatementInsertRowLimitProbes, oceanbasev1review.StatementInsertRowLimit, code.InsertTooManyRows, "inserts"))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, oceanbaseRowLimitAdvisor(oceanbasev1review.StatementAffectedRowLimitProbes, oceanbasev1review.StatementAffectedRowLimit, code.StatementAffectedRowExceedsLimit, "affected (estimated)"))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, advisor.NewMySQLDMLDryRunAdvisor(extract, byteOffsetLine, oceanbasev1review.StatementDMLDryRunProbes, oceanbasev1review.StatementDMLDryRun))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_OFFLINE_DDL, advisor.Func(disallowOfflineDDL))

	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, f.ColumnAutoIncrementInitialValue(oceanbasev1review.ColumnAutoIncrementInitialValue))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, f.ColumnAutoIncrementMustInteger(oceanbasev1review.ColumnAutoIncrementMustInteger))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, f.ColumnAutoIncrementMustUnsigned(oceanbasev1review.ColumnAutoIncrementMustUnsigned))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_COMMENT, f.ColumnComment(oceanbasev1review.ColumnComment))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, f.ColumnCurrentTimeCountLimit(oceanbasev1review.ColumnCurrentTimeCountLimit))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE, f.ColumnDisallowChange(oceanbasev1review.ColumnDisallowChange))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, f.ColumnDisallowChangeType(oceanbasev1review.ColumnDisallowChangeType))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, f.ColumnDisallowChangingOrder(oceanbasev1review.ColumnDisallowChangingOrder))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP, f.ColumnDisallowDrop(oceanbasev1review.ColumnDisallowDrop))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, f.ColumnDisallowDropInIndex(oceanbasev1review.ColumnDisallowDropInIndex))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, f.ColumnDisallowSetCharset(oceanbasev1review.ColumnDisallowSetCharset))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, f.ColumnMaximumCharacterLength(oceanbasev1review.ColumnMaximumCharacterLength))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, f.ColumnMaximumVarcharLength(oceanbasev1review.ColumnMaximumVarcharLength))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_NO_NULL, f.ColumnNoNull(oceanbasev1review.ColumnNoNull))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, f.ColumnRequireDefault(oceanbasev1review.ColumnRequireDefault))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_REQUIRED, f.ColumnRequired(oceanbasev1review.ColumnRequired))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, f.ColumnSetDefaultForNotNull(oceanbasev1review.ColumnSetDefaultForNotNull))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, f.ColumnTypeDisallowList(oceanbasev1review.ColumnTypeDisallowList))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, f.DatabaseDropEmptyDatabase(oceanbasev1review.DatabaseDropEmptyDatabase))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, f.IndexKeyNumberLimit(oceanbasev1review.IndexKeyNumberLimit))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, f.IndexNoDuplicateColumn(oceanbasev1review.IndexNoDuplicateColumn))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, f.IndexPKTypeLimit(oceanbasev1review.IndexPKTypeLimit))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, f.IndexPrimaryKeyTypeAllowlist(oceanbasev1review.IndexPrimaryKeyTypeAllowlist))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, f.IndexTotalNumberLimit(oceanbasev1review.IndexTotalNumberLimit))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, f.IndexTypeNoBlob(oceanbasev1review.IndexTypeNoBlob))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, f.NamingAutoIncrementColumn(oceanbasev1review.NamingAutoIncrementColumn))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_COLUMN, f.NamingColumn(oceanbasev1review.NamingColumn))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_INDEX_FK, f.NamingForeignKey(oceanbasev1review.NamingForeignKey))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_INDEX_IDX, f.NamingIndex(oceanbasev1review.NamingIndex))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_TABLE, f.NamingTable(oceanbasev1review.NamingTable))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_INDEX_UK, f.NamingUniqueKey(oceanbasev1review.NamingUniqueKey))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, f.SchemaBackwardCompatibility(oceanbasev1review.SchemaBackwardCompatibility))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_ADD_COLUMN_WITHOUT_POSITION, f.StatementAddColumnWithoutPosition(oceanbasev1review.StatementAddColumnWithoutPosition))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, f.StatementDisallowCommit(oceanbasev1review.StatementDisallowCommit))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, f.StatementDisallowLimit(oceanbasev1review.StatementDisallowLimit))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, f.StatementDisallowOrderBy(oceanbasev1review.StatementDisallowOrderBy))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, f.StatementDisallowTruncate(oceanbasev1review.StatementDisallowTruncate))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, f.StatementInsertDisallowOrderByRand(oceanbasev1review.StatementInsertDisallowOrderByRand))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, f.StatementInsertMustSpecifyColumn(oceanbasev1review.StatementInsertMustSpecifyColumn))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, f.StatementMergeAlterTable(oceanbasev1review.StatementMergeAlterTable))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, f.StatementSelectNoSelectAll(oceanbasev1review.StatementSelectNoSelectAll))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, f.StatementWhereNoLeadingWildcardLike(oceanbasev1review.StatementWhereNoLeadingWildcardLike))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, f.StatementWhereRequireSelect(oceanbasev1review.StatementWhereRequireSelect))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, f.StatementWhereRequireUpdateDelete(oceanbasev1review.StatementWhereRequireUpdateDelete))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, f.SystemCharsetAllowlist(oceanbasev1review.SystemCharsetAllowlist))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, f.SystemCollationAllowlist(oceanbasev1review.SystemCollationAllowlist))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_COMMENT, f.TableComment(oceanbasev1review.TableComment))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, f.TableDisallowPartition(oceanbasev1review.TableDisallowPartition))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, f.TableDropNaming(oceanbasev1review.TableDropNaming))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, f.TableNoForeignKey(oceanbasev1review.TableNoForeignKey))
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_REQUIRE_PK, f.TableRequirePK(oceanbasev1review.TableRequirePK))
}

func disallowOfflineDDL(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	lookup, err := advisor.ColumnExtraOf(checkCtx)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range advisor.ExtractNodes(checkCtx.ParsedStatements, oceanbaseparser.GetNode) {
		nodes := []ast.Node{stmt.Node}
		extras := make(map[v1review.ColumnRef]string)
		for _, ref := range oceanbasev1review.StatementDisallowOfflineDDLLookups(nodes, checkCtx.CurrentDatabase) {
			if extra, err := lookup.ColumnExtra(ctx, ref.Database, ref.Table, ref.Column); err == nil {
				extras[ref] = extra
			}
		}
		for _, f := range oceanbasev1review.StatementDisallowOfflineDDL(nodes, checkCtx.CurrentDatabase, extras) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementOfflineDDL.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("%s is an offline DDL operation.", f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(findingLineOf(stmt, f)),
			})
		}
	}
	return adviceList, nil
}

func oceanbaseRowLimitAdvisor(
	probesFn func([]ast.Node) []int,
	judgeFn func([]ast.Node, map[int]v1review.Probe, int) []v1review.Finding,
	exceededCode code.Code,
	verb string,
) advisor.Func {
	return func(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		numberPayload := checkCtx.Rule.GetNumberPayload()
		if numberPayload == nil {
			return nil, errors.New("number_payload is required for this rule")
		}
		maxRow := int(numberPayload.Number)
		title := checkCtx.Rule.Type.String()

		stmts := advisor.ExtractNodes(checkCtx.ParsedStatements, oceanbaseparser.GetNode)
		nodes := make([]ast.Node, len(stmts))
		for i, s := range stmts {
			nodes[i] = s.Node
		}

		var explains advisor.ExplainBudget
		probes := make(map[int]v1review.Probe)
		for _, i := range probesFn(nodes) {
			if !explains.Spend(nodePosition(stmts[i], nodes[i])) {
				continue
			}
			probe, err := explainRows(ctx, checkCtx, stmts[i].Text)
			if err != nil {
				return nil, err
			}
			probes[i] = probe
		}

		var adviceList []*storepb.Advice
		for _, f := range judgeFn(nodes, probes, maxRow) {
			stmt := stmts[f.Statement]
			text := stmt.Text
			adviceCode, content := exceededCode, ""
			switch f.Code {
			case v1review.DryRunFailed:
				content = fmt.Sprintf("\"%s\" dry runs failed: %s", text, f.Value)
			case v1review.RowCountFailed:
				adviceCode, content = code.Internal, fmt.Sprintf("failed to get row count for \"%s\": %s", text, f.Value)
			case v1review.RowLimitExceeded:
				if exceededCode == code.StatementAffectedRowExceedsLimit {
					content = fmt.Sprintf("\"%s\" affected %s rows (estimated). The count exceeds %d.", text, f.Value, maxRow)
				} else {
					content = fmt.Sprintf("\"%s\" %s %s rows. The count exceeds %d.", text, verb, f.Value, maxRow)
				}
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          adviceCode.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: base.ConvertANTLRLineToPosition(findingLineOf(stmt, f)),
			})
		}
		return explains.AppendSkippedAdvice(adviceList, title, exceededCode), nil
	}
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

func findingLineOf(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) int {
	loc := ast.Loc{Start: f.Start, End: f.End}
	if f.Start == 0 && f.End == 0 {
		loc.Start = -1
	}
	return lineOf(stmt.BaseLine, stmt.Text, loc)
}

func nodePosition(stmt advisor.V1Stmt[ast.Node], node ast.Node) *storepb.Position {
	loc := ast.Loc{Start: -1}
	switch n := node.(type) {
	case *ast.InsertStmt:
		loc = n.Loc
	case *ast.UpdateStmt:
		loc = n.Loc
	case *ast.DeleteStmt:
		loc = n.Loc
	default:
	}
	return base.ConvertANTLRLineToPosition(lineOf(stmt.BaseLine, stmt.Text, loc))
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
