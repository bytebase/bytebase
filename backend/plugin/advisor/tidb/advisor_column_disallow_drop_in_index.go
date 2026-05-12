package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnDisallowDropInIndexAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, &ColumnDisallowDropInIndexAdvisor{})
}

// ColumnDisallowDropInIndexAdvisor is the advisor checking for disallow DROP COLUMN in index.
type ColumnDisallowDropInIndexAdvisor struct{}

// Check tracks index-column membership across the reviewed statements
// (Recipe A cross-stmt state) and emits an advice when an ALTER TABLE
// DROP COLUMN targets a column that's part of an index.
//
// Per-arm state-mutation semantics (cumulative #25 audit axis):
//   - CreateTableStmt: INCREMENT — populate tables[name][col]=true for
//     each plain column in the table's KEY/INDEX constraints (omni
//     ConstrIndex; pingcap analog was ConstraintIndex, plain non-unique
//     only — UNIQUE / PRIMARY KEY / FULLTEXT / SPATIAL are NOT tracked).
//   - AlterTableStmt ATDropColumn: SIDE-LOAD CATALOG + READ — populate
//     tables[name] from OriginalMetadata.ListIndexes() (which reflects
//     the FINAL schema state including any pre-statement indexes the
//     reviewed CREATE TABLEs didn't add), then check if the dropped
//     column is in the index set. Side-load is idempotent across
//     multiple DROP COLUMN cmds in the same ALTER.
//
// Identifier case-sensitivity (cumulative #19): pre-omni used `.O`
// (original case) throughout; omni's direct strings preserve user case.
// `CREATE TABLE t(A INT, INDEX(A)); ALTER TABLE t DROP COLUMN a` does
// NOT fire (case-mismatched lookup misses the index set) — matches
// pingcap-tidb's pre-omni case-sensitive behavior. Pinned with fixture.
func (*ColumnDisallowDropInIndexAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	tables := make(tableState)

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table == nil {
				continue
			}
			tableName := n.Table.Name
			if tables[tableName] == nil {
				tables[tableName] = make(columnSet)
			}
			for _, c := range n.Constraints {
				if c == nil || c.Type != ast.ConstrIndex {
					continue
				}
				for _, col := range omniIndexColumns(c.IndexColumns) {
					tables[tableName][col] = true
				}
			}
		case *ast.AlterTableStmt:
			if n.Table == nil {
				continue
			}
			tableName := n.Table.Name
			stmtLine := ostmt.FirstTokenLine()
			for _, cmd := range n.Commands {
				if cmd == nil || cmd.Type != ast.ATDropColumn {
					continue
				}
				if tableMetadata := checkCtx.OriginalMetadata.GetSchemaMetadata("").GetTable(tableName); tableMetadata != nil {
					if tables[tableName] == nil {
						tables[tableName] = make(columnSet)
					}
					for _, idx := range tableMetadata.ListIndexes() {
						for _, indexedCol := range idx.GetProto().GetExpressions() {
							tables[tableName][indexedCol] = true
						}
					}
				}
				colName := cmd.Name
				if _, isIndexCol := tables[tableName][colName]; isIndexCol {
					adviceList = append(adviceList, &storepb.Advice{
						Status:        level,
						Code:          code.DropIndexColumn.Int32(),
						Title:         title,
						Content:       fmt.Sprintf("`%s`.`%s` cannot drop index column", tableName, colName),
						StartPosition: common.ConvertANTLRLineToPosition(stmtLine),
					})
				}
			}
		default:
		}
	}
	return adviceList, nil
}
