package tidb

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
	f := advisor.NewMySQLFamily(advisor.MySQLFamilyOptions[ast.Node]{
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
	})

	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK, &advisor.BuiltinWalkThroughCheckAdvisor{})
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, advisor.Func(tidbPriorBackup))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, advisor.Func(tidbDMLDryRun))

	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, f.ColumnAutoIncrementInitialValue(tidbv1review.ColumnAutoIncrementInitialValue))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, f.ColumnAutoIncrementMustInteger(tidbv1review.ColumnAutoIncrementMustInteger))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, f.ColumnAutoIncrementMustUnsigned(tidbv1review.ColumnAutoIncrementMustUnsigned))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_COMMENT, f.ColumnComment(tidbv1review.ColumnComment))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, f.ColumnCurrentTimeCountLimit(tidbv1review.ColumnCurrentTimeCountLimit))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE, f.ColumnDisallowChange(tidbv1review.ColumnDisallowChange))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, f.ColumnDisallowChangeType(tidbv1review.ColumnDisallowChangeType))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, f.ColumnDisallowChangingOrder(tidbv1review.ColumnDisallowChangingOrder))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, f.ColumnDisallowDropInIndex(tidbv1review.ColumnDisallowDropInIndex))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, f.ColumnDisallowSetCharset(tidbv1review.ColumnDisallowSetCharset))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, f.ColumnMaximumCharacterLength(tidbv1review.ColumnMaximumCharacterLength))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_NO_NULL, f.ColumnNoNull(tidbv1review.ColumnNoNull))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, f.ColumnRequireDefault(tidbv1review.ColumnRequireDefault))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_REQUIRED, f.ColumnRequired(tidbv1review.ColumnRequired))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, f.ColumnSetDefaultForNotNull(tidbv1review.ColumnSetDefaultForNotNull))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, f.ColumnTypeDisallowList(tidbv1review.ColumnTypeDisallowList))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, f.DatabaseDropEmptyDatabase(tidbv1review.DatabaseDropEmptyDatabase))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, f.IndexKeyNumberLimit(tidbv1review.IndexKeyNumberLimit))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, f.IndexNoDuplicateColumn(tidbv1review.IndexNoDuplicateColumn))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, f.IndexPKTypeLimit(tidbv1review.IndexPKTypeLimit))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, f.IndexPrimaryKeyTypeAllowlist(tidbv1review.IndexPrimaryKeyTypeAllowlist))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, f.IndexTotalNumberLimit(tidbv1review.IndexTotalNumberLimit))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, f.IndexTypeNoBlob(tidbv1review.IndexTypeNoBlob))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, f.NamingAutoIncrementColumn(tidbv1review.NamingAutoIncrementColumn))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_COLUMN, f.NamingColumn(tidbv1review.NamingColumn))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_INDEX_FK, f.NamingForeignKey(tidbv1review.NamingForeignKey))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_INDEX_IDX, f.NamingIndex(tidbv1review.NamingIndex))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_TABLE, f.NamingTable(tidbv1review.NamingTable))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_INDEX_UK, f.NamingUniqueKey(tidbv1review.NamingUniqueKey))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, f.SchemaBackwardCompatibility(tidbv1review.SchemaBackwardCompatibility))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, f.StatementDisallowCommit(tidbv1review.StatementDisallowCommit))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, f.StatementDisallowLimit(tidbv1review.StatementDisallowLimit))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, f.StatementDisallowOrderBy(tidbv1review.StatementDisallowOrderBy))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, f.StatementInsertDisallowOrderByRand(tidbv1review.StatementInsertDisallowOrderByRand))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, f.StatementInsertMustSpecifyColumn(tidbv1review.StatementInsertMustSpecifyColumn))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, f.StatementMergeAlterTable(tidbv1review.StatementMergeAlterTable))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, f.StatementSelectNoSelectAll(tidbv1review.StatementSelectNoSelectAll))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, f.StatementWhereNoLeadingWildcardLike(tidbv1review.StatementWhereNoLeadingWildcardLike))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, f.StatementWhereRequireSelect(tidbv1review.StatementWhereRequireSelect))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, f.StatementWhereRequireUpdateDelete(tidbv1review.StatementWhereRequireUpdateDelete))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, f.SystemCharsetAllowlist(tidbv1review.SystemCharsetAllowlist))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, f.SystemCollationAllowlist(tidbv1review.SystemCollationAllowlist))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_COMMENT, f.TableComment(tidbv1review.TableComment))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, f.TableDisallowPartition(tidbv1review.TableDisallowPartition))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, f.TableDropNaming(tidbv1review.TableDropNaming))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, f.TableNoForeignKey(tidbv1review.TableNoForeignKey))
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_REQUIRE_PK, f.TableRequirePK(tidbv1review.TableRequirePK))
}

func tidbPriorBackup(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup {
		return nil, nil
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	defaultDB := checkCtx.DBSchema.GetName()
	if defaultDB == "" {
		defaultDB = checkCtx.CurrentDatabase
	}
	databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_TIDB)
	schema := v1review.Schema{Metadata: checkCtx.DBSchema, ObjectCaseSensitive: checkCtx.IsObjectCaseSensitive}
	facts := v1review.PriorBackupFacts{
		TotalSize:      checkCtx.StatementsTotalSize,
		MaxSize:        common.MaxSheetCheckSize,
		Database:       databaseName,
		DatabaseExists: advisor.DatabaseExists(ctx, checkCtx, databaseName),
	}
	findings := tidbv1review.PriorBackupCheck(stmtNodes(stmts), defaultDB, schema, facts)
	return advisor.RenderPriorBackupFindings(checkCtx.Rule.Type.String(), level, findings, func(idx int) int {
		return stmts[idx].FirstTokenLine()
	}), nil
}

func tidbDMLDryRun(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	if advisor.ContainsDDL(checkCtx.DBType, checkCtx.ParsedStatements) {
		return nil, nil
	}
	prober, err := advisor.ProbeFor(checkCtx)
	if err != nil {
		return nil, err
	}

	nodes := stmtNodes(stmts)
	probes := make(map[int]v1review.Probe)
	for n, i := range tidbv1review.StatementDMLDryRunProbes(nodes) {
		if reason, failed := probeTiDBDML(ctx, prober, stmts[i]); failed {
			probes[i] = v1review.Probe{Error: reason}
		} else {
			probes[i] = v1review.Probe{}
		}
		if n+1 >= common.MaximumLintExplainSize {
			break
		}
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range tidbv1review.StatementDMLDryRun(nodes, probes) {
		ostmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.StatementDMLDryRunFailed.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", ostmt.TrimmedText(), f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
		})
	}
	return adviceList, nil
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
				return slices.ContainsFunc(addColumnTargets(cmd), func(col *ast.ColumnDef) bool { return col != nil })
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
	if !ok || alter.Table == nil {
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
