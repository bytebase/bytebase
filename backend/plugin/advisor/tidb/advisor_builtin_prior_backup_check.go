package tidb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

const (
	maxMixedDMLCount = 5
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &StatementPriorBackupCheckAdvisor{})
}

// StatementPriorBackupCheckAdvisor flags inputs incompatible with the
// prior-backup workflow: mixed DDL/DML, > maxMixedDMLCount UPDATE+DELETE
// statements unless all UPDATEs target a single table with a unique-
// column predicate, and missing backup database.
//
// Mechanical port of pre-omni behavior (invariant #7). The tidb logic
// here is OLDER than mysql's modernized prior_backup_check (which uses
// per-table DML-mixing detection via mysqlparser.ExtractTables). Future
// alignment is a Phase 2 feature ticket; this batch preserves pre-omni
// tidb behavior exactly.
//
// Audit axes applied:
//   - #19 (case-sensitivity): pre-omni used .L lowercase on Column/
//     Table/Schema names. Omni preserves user case via direct strings;
//     explicit strings.ToLower applied at lookup sites.
//   - #26 (UNION-root): omni unifies UNION-rooted UpdateStmt sources
//     into SelectStmt-with-SetOp. The extractor returns nil for such
//     cases (matching pre-omni's *SetOprStmt arm).
//   - #29 (filter-effect mismatch): isConstantLit enumerates omni's
//     literal types (IntLit/StringLit/...) — pre-omni's `ast.ValueExpr`
//     interface check.
type StatementPriorBackupCheckAdvisor struct{}

// Check evaluates the prior-backup compatibility of the reviewed
// statements. Gated on `checkCtx.EnablePriorBackup`.
func (*StatementPriorBackupCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup {
		return nil, nil
	}

	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	var updateStatements []*ast.UpdateStmt
	var deleteStatements []*ast.DeleteStmt

	for _, ostmt := range stmts {
		node := ostmt.Node
		if omniIsDDLStmt(node) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       "Prior backup cannot deal with mixed DDL and DML statements",
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: common.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
			})
		}
		if u, ok := node.(*ast.UpdateStmt); ok {
			updateStatements = append(updateStatements, u)
		}
		if d, ok := node.(*ast.DeleteStmt); ok {
			deleteStatements = append(deleteStatements, d)
		}
	}

	databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_TIDB)
	if !advisor.DatabaseExists(ctx, checkCtx, databaseName) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("Prior backup check failed: need database %q to do prior backup but it does not exist", databaseName),
			Code:          code.BuiltinPriorBackupCheck.Int32(),
			StartPosition: nil,
		})
	}

	if len(updateStatements)+len(deleteStatements) > maxMixedDMLCount && !omniUpdateForOneTableWithUnique(checkCtx.DBSchema, updateStatements, deleteStatements) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("Prior backup is feasible only with up to %d statements that are either UPDATE or DELETE, or if all UPDATEs target the same table with a PRIMARY or UNIQUE KEY in the WHERE clause", maxMixedDMLCount),
			Code:          code.BuiltinPriorBackupCheck.Int32(),
			StartPosition: nil,
		})
	}

	return adviceList, nil
}

// omniUpdateForOneTableWithUnique returns true when ALL UPDATEs in the
// batch target the same single table AND each has a unique-column
// equality predicate in its WHERE clause. Used by the
// >maxMixedDMLCount short-circuit. Any DELETE present disqualifies.
func omniUpdateForOneTableWithUnique(dbMetadata *storepb.DatabaseSchemaMetadata, updates []*ast.UpdateStmt, deletes []*ast.DeleteStmt) bool {
	if len(deletes) > 0 {
		return false
	}

	var table *priorBackupTable
	for _, update := range updates {
		tables, err := omniExtractUpdateTables(update.Tables)
		if err != nil {
			slog.Debug("failed to extract update table reference", log.BBError(err))
			return false
		}
		if len(tables) != 1 {
			return false
		}
		if table == nil {
			table = &tables[0]
		} else if !equalPriorBackupTable(table, &tables[0]) {
			return false
		}
		if !omniHasUniqueInWhereClause(dbMetadata, update, table) {
			return false
		}
	}
	return true
}

// omniHasUniqueInWhereClause returns true when the UPDATE's WHERE
// clause has equality predicates covering all columns of some unique
// or primary index on the target table (read from dbMetadata).
func omniHasUniqueInWhereClause(dbMetadata *storepb.DatabaseSchemaMetadata, update *ast.UpdateStmt, table *priorBackupTable) bool {
	if update.Where == nil {
		return false
	}
	list := omniExtractColumnsInEqualCondition(table, update.Where)
	columnMap := make(map[string]bool)
	for _, column := range list {
		// Cumulative #19: omni preserves user case; lowercase for
		// case-insensitive index-column matching (pre-omni used
		// `.L` access which was implicitly lowercase).
		columnMap[strings.ToLower(column)] = true
	}
	if dbMetadata == nil {
		return false
	}
	for _, schema := range dbMetadata.Schemas {
		for _, tableSchema := range schema.Tables {
			if !strings.EqualFold(tableSchema.Name, table.table) {
				continue
			}
			for _, index := range tableSchema.Indexes {
				if !index.Unique && !index.Primary {
					continue
				}
				covered := true
				for _, column := range index.Expressions {
					if !columnMap[strings.ToLower(column)] {
						covered = false
						break
					}
				}
				if covered {
					return true
				}
			}
		}
	}
	return false
}

// omniExtractColumnsInEqualCondition walks the WHERE expression tree
// and returns the column names that appear in equality predicates with
// a literal/constant on the other side. Cumulative #29 family —
// preserves pre-omni's behavior of only matching the column in
// `col = literal` predicates linked by AND. Recursive descent over
// AND chains; single equality drives the leaf return.
func omniExtractColumnsInEqualCondition(table *priorBackupTable, node ast.ExprNode) []string {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.BinaryExpr:
		switch n.Op {
		case ast.BinOpAnd:
			return append(
				omniExtractColumnsInEqualCondition(table, n.Left),
				omniExtractColumnsInEqualCondition(table, n.Right)...,
			)
		case ast.BinOpEq:
			if omniIsConstantLit(n.Right) {
				return omniExtractColumnsInEqualCondition(table, n.Left)
			}
			if omniIsConstantLit(n.Left) {
				return omniExtractColumnsInEqualCondition(table, n.Right)
			}
			return nil
		default:
			return nil
		}
	case *ast.ColumnRef:
		// Cumulative #19: omni preserves user case in Schema/Table/
		// Column. Pre-omni used .L (lowercase); we explicitly EqualFold
		// for case-insensitive table/schema scoping.
		if n.Schema != "" && table.database != "" && !strings.EqualFold(n.Schema, table.database) {
			return nil
		}
		if n.Table != "" && table.table != "" && !strings.EqualFold(n.Table, table.table) {
			return nil
		}
		// Pre-omni returned `n.Name.Name.L` (lowercase); we return the
		// original-case Column and let callers (omniHasUniqueInWhereClause)
		// lowercase for map keying.
		return []string{n.Column}
	default:
		return nil
	}
}

// omniIsConstantLit reports whether the given expression is a literal
// value (pre-omni `ast.ValueExpr` interface check). Omni splits
// literals into 8 concrete types — enumerated here. Any future omni
// literal type would need to be added; helper unit tests cover the
// 8 known types + ColumnRef + nil + BinaryExpr negatives.
func omniIsConstantLit(expr ast.ExprNode) bool {
	if expr == nil {
		return false
	}
	switch expr.(type) {
	case *ast.IntLit, *ast.StringLit, *ast.FloatLit, *ast.BoolLit,
		*ast.NullLit, *ast.HexLit, *ast.BitLit, *ast.TemporalLit:
		return true
	default:
		return false
	}
}

// priorBackupTable is the (database, table) qualifier captured from
// UPDATE's table reference. Kept file-local — only this advisor uses it.
type priorBackupTable struct {
	database string
	table    string
}

func equalPriorBackupTable(t1, t2 *priorBackupTable) bool {
	if t1 == nil || t2 == nil {
		return false
	}
	return t1.database == t2.database && t1.table == t2.table
}

// omniExtractUpdateTables walks the omni UpdateStmt.Tables slice
// (each element is a TableExpr — *TableRef, *JoinClause, or
// *SubqueryExpr). Returns the set of base-table references reached.
//
// Cumulative #26 (UNION-root): omni unifies UNION-rooted derived
// tables into SelectStmt-with-SetOp accessed via SubqueryExpr.Select.
// We do NOT descend into derived-table SELECTs — pre-omni's
// extractResultSetNode returned nil for *SelectStmt / *SubqueryExpr /
// *SetOprStmt. Preserved.
func omniExtractUpdateTables(tables []ast.TableExpr) ([]priorBackupTable, error) {
	var result []priorBackupTable
	for _, t := range tables {
		extracted, err := omniExtractTableExpr(t)
		if err != nil {
			return nil, err
		}
		result = append(result, extracted...)
	}
	return result, nil
}

// omniExtractTableExpr walks a single omni TableExpr, returning the
// base-table references reached. Mirrors pre-omni extractResultSetNode
// dispatch — base tables yield (db, table); derived tables (SELECT
// subqueries, UNION-rooted SelectStmt) yield nothing (per pre-omni).
func omniExtractTableExpr(t ast.TableExpr) ([]priorBackupTable, error) {
	if t == nil {
		return nil, nil
	}
	switch n := t.(type) {
	case *ast.TableRef:
		return []priorBackupTable{{
			table:    n.Name,
			database: n.Schema,
		}}, nil
	case *ast.JoinClause:
		left, err := omniExtractTableExpr(n.Left)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract left node in join")
		}
		right, err := omniExtractTableExpr(n.Right)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract right node in join")
		}
		return append(left, right...), nil
	case *ast.SubqueryExpr:
		// Pre-omni returned nil for *SubqueryExpr. Preserved — derived
		// tables aren't unique-base-table candidates. Cumulative #26.
		return nil, nil
	default:
		// Pre-omni returned nil for *SelectStmt / *SetOprStmt arms.
		// Omni's UNION-rooted SelectStmt-with-SetOp is reached only
		// inside a SubqueryExpr at this layer; the *SubqueryExpr arm
		// above handles it. Other TableExpr concrete types (if any
		// are added in future omni grammar evolution) fall through
		// to nil — same conservative pre-omni shape.
		return nil, nil
	}
}

// omniIsDDLStmt reports whether the given statement node is a DDL
// (schema-changing) statement. Pre-omni used `ast.DDLNode` marker
// interface; omni has no such interface, so we enumerate concrete
// DDL types. The list mirrors the union of types pingcap's DDLNode
// interface accepts, mapped to omni's concrete equivalents.
//
// Future-staleness note: if omni adds new DDL types (e.g., new
// CREATE/ALTER variants), this list needs updating. Test coverage
// via fixtures should catch obvious omissions; for now the list
// covers the DDL types pre-omni production traffic exercises.
func omniIsDDLStmt(node ast.Node) bool {
	switch node.(type) {
	case *ast.CreateTableStmt, *ast.AlterTableStmt, *ast.DropTableStmt,
		*ast.CreateIndexStmt, *ast.DropIndexStmt,
		*ast.CreateViewStmt, *ast.DropViewStmt,
		*ast.CreateDatabaseStmt, *ast.AlterDatabaseStmt, *ast.DropDatabaseStmt,
		*ast.TruncateStmt, *ast.RenameTableStmt,
		*ast.CreateUserStmt, *ast.DropUserStmt, *ast.AlterUserStmt,
		*ast.CreateRoleStmt, *ast.DropRoleStmt,
		*ast.CreateFunctionStmt, *ast.CreateTriggerStmt, *ast.CreateEventStmt,
		*ast.CreatePlacementPolicyStmt, *ast.AlterPlacementPolicyStmt, *ast.DropPlacementPolicyStmt,
		*ast.CreateTablespaceStmt, *ast.AlterTablespaceStmt, *ast.DropTablespaceStmt,
		*ast.CreateServerStmt:
		return true
	default:
		return false
	}
}
