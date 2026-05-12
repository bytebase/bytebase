package tidb

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequirementAdvisor{})
}

// ColumnRequirementAdvisor is the advisor checking for column requirement.
type ColumnRequirementAdvisor struct{}

// Check tracks per-table required-column presence across the reviewed
// statements and emits a per-table advice listing any required columns
// that ended up missing.
//
// Per-arm state-mutation semantics (cumulative #25 audit axis — five
// distinct semantics across arms; preserve each independently):
//   - CreateTableStmt: REPLACE — `initEmptyTable(name)` then `addColumn`
//     per column in the new table. Resets prior state on re-creation.
//   - DropTableStmt: DELETE — `delete(tables, name)` per dropped table.
//     The table no longer appears in `generateAdviceList`. Pre-omni did
//     NOT delete from the `line` map; preserved (orphaned line entries
//     are harmless — generateAdviceList iterates tables, not line).
//   - AlterTableStmt ATRenameColumn: READ-MODIFY-WRITE — `renameColumn`
//     toggles old→false / new→true in the required-column map; lazy-
//     initializes the table state as "all required present" if absent.
//   - AlterTableStmt ATAddColumn: READ-MODIFY-WRITE — `addColumn` per
//     added column (handles both singular `cmd.Column` and grouped
//     `cmd.Columns` via `addColumnTargets`); lazy-initializes the
//     table state as "all required present" if absent. NO line update
//     (ADD COLUMN can only make missing required columns present;
//     never triggers a new advice).
//   - AlterTableStmt ATDropColumn: READ-MODIFY-WRITE — `dropColumn`
//     marks the column false if it's required.
//   - AlterTableStmt ATChangeColumn: equivalent to RENAME — `cmd.Name`
//     is old, `cmd.Column.Name` is new (verified in
//     advisor_column_disallow_changing_type.go pattern).
//
// Line tracking has per-arm semantics distinct from primary state
// (cumulative #27 — auxiliary state maps have their own conditionality
// contracts; mechanical port must audit each state map independently):
//   - CreateTable: seeds line[table] unconditionally.
//   - DropTable: does NOT delete from line (orphaned entries harmless;
//     generateAdviceList iterates tables, not line).
//   - ATRenameColumn: updates line UNCONDITIONALLY (RENAME is a more
//     explicit "user modified this table at this line" signal).
//   - ATAddColumn: does NOT update line (ADD can only make missing
//     required columns present; never triggers new advice).
//   - ATDropColumn: updates line ONLY when the dropped column was
//     required (return value of dropColumn).
//   - ATChangeColumn: updates line ONLY when the old name was required
//     (return value of renameColumn). Asymmetric vs ATRenameColumn —
//     pre-omni intentional asymmetry preserved.
//
// Identifier case-sensitivity (cumulative #19): pre-omni used `.O`
// throughout (no `.L` lowercase); omni's direct strings preserve user
// case. `CREATE TABLE t(A INT); ALTER TABLE t RENAME COLUMN a TO X`
// matches case-sensitively — pre-omni would NOT match `a` to required
// column `A`. Preserved.
func (*ColumnRequirementAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for column required rule")
	}
	requiredColumns := newColumnSet(stringArrayPayload.List)
	title := checkCtx.Rule.Type.String()

	v := &columnRequirementState{
		requiredColumns: requiredColumns,
		tables:          make(tableState),
		line:            make(map[string]int),
	}

	for _, ostmt := range stmts {
		stmtLine := ostmt.FirstTokenLine()
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table == nil {
				continue
			}
			v.line[n.Table.Name] = stmtLine
			v.initEmptyTable(n.Table.Name)
			for _, column := range n.Columns {
				if column == nil {
					continue
				}
				v.addColumn(n.Table.Name, column.Name)
			}
		case *ast.DropTableStmt:
			for _, table := range n.Tables {
				if table != nil {
					delete(v.tables, table.Name)
				}
			}
		case *ast.AlterTableStmt:
			if n.Table == nil {
				continue
			}
			table := n.Table.Name
			for _, cmd := range n.Commands {
				if cmd == nil {
					continue
				}
				switch cmd.Type {
				case ast.ATRenameColumn:
					// Cumulative #27: pre-omni updated line[table]
					// UNCONDITIONALLY for RENAME COLUMN, even when
					// neither old nor new is in the required set.
					// CHANGE COLUMN (below) is conditional on return
					// value — asymmetric pre-omni behavior preserved.
					// RENAME is a more explicit "user modified this
					// table at this line" signal that should mark the
					// position regardless of required-column impact.
					v.renameColumn(table, cmd.Name, cmd.NewName)
					v.line[table] = stmtLine
				case ast.ATAddColumn:
					for _, column := range addColumnTargets(cmd) {
						if column == nil {
							continue
						}
						v.addColumn(table, column.Name)
					}
				case ast.ATDropColumn:
					if v.dropColumn(table, cmd.Name) {
						v.line[table] = stmtLine
					}
				case ast.ATChangeColumn:
					if cmd.Column == nil {
						continue
					}
					if v.renameColumn(table, cmd.Name, cmd.Column.Name) {
						v.line[table] = stmtLine
					}
				default:
				}
			}
		default:
		}
	}

	return v.generateAdviceList(level, title), nil
}

type columnRequirementState struct {
	requiredColumns columnSet
	tables          tableState
	line            map[string]int
}

func (v *columnRequirementState) generateAdviceList(level storepb.Advice_Status, title string) []*storepb.Advice {
	var adviceList []*storepb.Advice
	for _, tableName := range v.tables.tableList() {
		table := v.tables[tableName]
		var missingColumns []string
		for column := range v.requiredColumns {
			if exist, ok := table[column]; !ok || !exist {
				missingColumns = append(missingColumns, column)
			}
		}
		if len(missingColumns) > 0 {
			slices.Sort(missingColumns)
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NoRequiredColumn.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Table `%s` requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
				StartPosition: common.ConvertANTLRLineToPosition(v.line[tableName]),
			})
		}
	}
	return adviceList
}

// initEmptyTable initializes a table with no required columns present.
func (v *columnRequirementState) initEmptyTable(name string) columnSet {
	v.tables[name] = make(columnSet)
	return v.tables[name]
}

// initFullTable initializes a table with all required columns present.
// Used for "we don't retrospectively check": when ALTER targets a table
// we haven't seen CREATE for, we assume it had all required columns,
// so only modifications surfaced in this review affect the verdict.
func (v *columnRequirementState) initFullTable(name string) columnSet {
	table := v.initEmptyTable(name)
	for column := range v.requiredColumns {
		table[column] = true
	}
	return table
}

// renameColumn marks the old name absent and the new name present, if
// either is in the required set. Returns true if the OLD name was
// required (meaning the rename could surface a new missing-required;
// caller updates the line).
func (v *columnRequirementState) renameColumn(table, oldColumn, newColumn string) bool {
	_, oldNeed := v.requiredColumns[oldColumn]
	_, newNeed := v.requiredColumns[newColumn]
	if !oldNeed && !newNeed {
		return false
	}
	t, ok := v.tables[table]
	if !ok {
		t = v.initFullTable(table)
	}
	if oldNeed {
		t[oldColumn] = false
	}
	if newNeed {
		t[newColumn] = true
	}
	return oldNeed
}

// dropColumn marks the column absent if it's required. Returns true
// when the dropped column was required.
func (v *columnRequirementState) dropColumn(table, column string) bool {
	if _, ok := v.requiredColumns[column]; !ok {
		return false
	}
	t, ok := v.tables[table]
	if !ok {
		t = v.initFullTable(table)
	}
	t[column] = false
	return true
}

// addColumn marks the column present if it's required.
func (v *columnRequirementState) addColumn(table, column string) {
	if _, ok := v.requiredColumns[column]; !ok {
		return
	}
	if t, ok := v.tables[table]; !ok {
		v.initFullTable(table)
	} else {
		t[column] = true
	}
}
