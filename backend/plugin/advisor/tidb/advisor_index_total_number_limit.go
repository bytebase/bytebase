package tidb

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*IndexTotalNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, &IndexTotalNumberLimitAdvisor{})
}

// IndexTotalNumberLimitAdvisor flags tables whose total index count
// (read from FinalMetadata) exceeds the configured maximum, gated on
// the reviewed statements actually touching the table with an
// index-creating operation.
type IndexTotalNumberLimitAdvisor struct{}

// Check tracks which tables had an index-creating operation across
// the reviewed statements (Recipe A; cross-stmt `lineForTable`
// aggregator), then for each such table queries FinalMetadata for
// the final index count.
//
// Per-arm state-mutation semantics (cumulative #25/#27 audit):
//   - CreateTableStmt: lineForTable[name] = stmtLine (UNCONDITIONAL —
//     creates the table itself; counts as a touch).
//   - CreateIndexStmt: lineForTable[name] = stmtLine (UNCONDITIONAL).
//   - ATAddColumn (inner loop over columns): if any new column has an
//     inline PRIMARY KEY or UNIQUE constraint → record and break
//     (FIRST-violating-column wins; subsequent index-creating columns
//     in the same grouped form don't add additional touches; pre-omni
//     `break` after `createIndex(column)` returned true preserved).
//   - ATAddConstraint: if constraint creates an index → record
//     (conditional; ConstrIndex / ConstrPrimaryKey / ConstrUnique /
//     ConstrFulltextIndex are index-creating; FK / Check / Spatial
//     are NOT, matching pre-omni scope).
//   - ATChangeColumn / ATModifyColumn: if new column def declares
//     inline PRIMARY KEY or UNIQUE → record (conditional).
//
// Cumulative #2 coverage: pingcap's ConstraintUniq / ConstraintUniqKey
// / ConstraintUniqIndex (3 distinct) unify under omni `ConstrUnique`;
// pingcap's `ConstraintKey` + `ConstraintIndex` unify under omni
// `ConstrIndex` (verified empirically). Single-arm port covers all
// 6 pingcap constraint forms mechanically.
func (*IndexTotalNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for index total number limit rule")
	}
	maximum := int(numberPayload.Number)
	title := checkCtx.Rule.Type.String()

	lineForTable := make(map[string]int)
	for _, ostmt := range stmts {
		stmtLine := ostmt.FirstTokenLine()
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table != nil {
				lineForTable[n.Table.Name] = stmtLine
			}
		case *ast.CreateIndexStmt:
			if n.Table != nil {
				lineForTable[n.Table.Name] = stmtLine
			}
		case *ast.AlterTableStmt:
			if n.Table == nil {
				continue
			}
			tableName := n.Table.Name
			for _, cmd := range n.Commands {
				if cmd == nil {
					continue
				}
				switch cmd.Type {
				case ast.ATAddColumn:
					if slices.ContainsFunc(addColumnTargets(cmd), omniColumnCreatesIndex) {
						lineForTable[tableName] = stmtLine
					}
				case ast.ATAddConstraint, ast.ATAddIndex:
					// Cumulative #17 sibling-parity convention: dual
					// arm preserved for forward-compat per established
					// pattern in utils.go collectIndexFamilyAlterTable.
					if omniConstraintCreatesIndex(cmd.Constraint) {
						lineForTable[tableName] = stmtLine
					}
				case ast.ATChangeColumn, ast.ATModifyColumn:
					if omniColumnCreatesIndex(cmd.Column) {
						lineForTable[tableName] = stmtLine
					}
				default:
				}
			}
		default:
		}
	}

	type tableEntry struct {
		name string
		line int
	}
	tableList := make([]tableEntry, 0, len(lineForTable))
	for name, line := range lineForTable {
		tableList = append(tableList, tableEntry{name: name, line: line})
	}
	slices.SortFunc(tableList, func(i, j tableEntry) int {
		switch {
		case i.line < j.line:
			return -1
		case i.line > j.line:
			return 1
		default:
			return 0
		}
	})

	var adviceList []*storepb.Advice
	for _, t := range tableList {
		schema := checkCtx.FinalMetadata.GetSchemaMetadata("")
		if schema == nil {
			continue
		}
		tableInfo := schema.GetTable(t.name)
		if tableInfo == nil {
			continue
		}
		if count := len(tableInfo.GetProto().Indexes); count > maximum {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.IndexCountExceedsLimit.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The count of index in table `%s` should be no more than %d, but found %d", t.name, maximum, count),
				StartPosition: common.ConvertANTLRLineToPosition(t.line),
			})
		}
	}
	return adviceList, nil
}

// omniConstraintCreatesIndex reports whether the given table-level
// constraint declares a new index. Mirrors pre-omni `createIndex(*ast.Constraint)`:
// PRIMARY KEY / UNIQUE (all 3 pingcap variants unified to ConstrUnique)
// / KEY+INDEX (both unified to ConstrIndex) / FULLTEXT KEY are
// index-creating. FOREIGN KEY and CHECK are NOT (pre-omni explicitly
// excluded). SPATIAL was also NOT in pre-omni's list — preserved.
func omniConstraintCreatesIndex(c *ast.Constraint) bool {
	if c == nil {
		return false
	}
	switch c.Type {
	case ast.ConstrPrimaryKey, ast.ConstrUnique, ast.ConstrIndex, ast.ConstrFulltextIndex:
		return true
	default:
		return false
	}
}

// omniColumnCreatesIndex reports whether the given column definition
// declares an inline PRIMARY KEY or UNIQUE constraint, which creates
// an index implicitly. Mirrors pre-omni's `createIndex(*ast.ColumnDef)`
// arm checking `ColumnOptionPrimaryKey` / `ColumnOptionUniqKey`. Omni
// represents column-level constraints as `column.Constraints
// []*ColumnConstraint` with `ColConstrPrimaryKey` / `ColConstrUnique`
// type values (verified empirically against parsenodes.go:404-412).
func omniColumnCreatesIndex(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	for _, c := range col.Constraints {
		if c == nil {
			continue
		}
		if c.Type == ast.ColConstrPrimaryKey || c.Type == ast.ColConstrUnique {
			return true
		}
	}
	return false
}
