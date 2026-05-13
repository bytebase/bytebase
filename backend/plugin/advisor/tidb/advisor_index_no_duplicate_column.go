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
	_ advisor.Advisor = (*IndexNoDuplicateColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, &IndexNoDuplicateColumnAdvisor{})
}

// IndexNoDuplicateColumnAdvisor flags index/constraint declarations
// where the same column appears twice in the key list.
type IndexNoDuplicateColumnAdvisor struct{}

// Check fires on PRIMARY KEY, UNIQUE (all 3 syntactic forms),
// INDEX/KEY, FOREIGN KEY constraints with duplicate plain-column
// entries. Real functional-index expressions (e.g., `((a + 1))`)
// are skipped — pre-omni's `if key.Expr == nil` filter and the
// omni port's `*ColumnRef`-only type-assert preserve the same
// author-intent contract ("skip non-column expressions to avoid
// name-based dedup on functional indexes"). Recipe A; no cross-
// stmt state.
//
// Cumulative #2 coverage (verified empirically): pingcap-tidb's
// parser produces `ConstraintUniq` (Tp=4) for ALL three UNIQUE
// syntactic forms — bare `UNIQUE`, `UNIQUE KEY`, and `UNIQUE INDEX`.
// `ConstraintUniqKey` (Tp=5) and `ConstraintUniqIndex` (Tp=6) are
// defined in pingcap's enum but the parser never produces them for
// these inputs. Pre-omni's defensive case list including all three
// was redundant (Tp=5/6 cases were unreachable). Omni unifies under
// `ConstrUnique`; the single-arm omni port matches pre-omni
// behavior mechanically — NO behavior change at the UNIQUE boundary.
// (Initial speculation framed this as a "silent UX improvement
// fixing a pre-omni miss of UniqKey" — empirical verification per
// invariant #9 disproved the speculation; ConstraintUniqKey is
// dead code in pingcap-tidb for these inputs.)
//
// Cumulative #28: PRIMARY KEY with the non-standard `PRIMARY KEY
// pk (cols)` syntax has empty Name in omni (parser drops it).
// Advice content uses `omniConstraintAdviceName` (utils.go) which
// falls back to "PRIMARY" canonical.
//
// Cumulative #29 (parser-quirk false-NEGATIVE silently fixed):
// pingcap-tidb's parser treats single-paren-wrapped column refs
// (e.g., `INDEX idx((a), (a))`) as expressions (`key.Expr != nil,
// key.Column == nil`). The pre-omni `if key.Expr == nil` filter
// (author intent: "skip non-column expressions") had a filter-
// effect that ALSO skipped paren-wrapped column refs as a side-
// effect — rule did NOT fire on `((a), (a))` despite the
// duplicate semantic being unambiguous. Omni follows MySQL 8.0
// spec: single-paren is grouping (flattened at parse time to
// inner ColumnRef); double-paren `((expr))` is the functional-
// index syntax. The `*ColumnRef`-only type-assert in
// `omniIndexColumns` correctly skips real functional indexes
// (`((a + 1))` → `*BinaryOperationExpr`, not ColumnRef) while
// catching paren-wrapped column duplicates. NOT a regression;
// inverse direction of cumulative #21 (#21 was a parser-quirk
// false-POSITIVE silently fixed; #29 is a parser-quirk false-
// NEGATIVE silently fixed). Both positive and negative scope-
// bounding fixtures pinned.
//
// Cumulative #17 sibling-parity: ATAddConstraint + ATAddIndex
// dual arm preserved per established convention.
func (*IndexNoDuplicateColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	type hit struct {
		tp     string
		table  string
		index  string
		column string
		line   int
	}
	var hits []hit

	for _, ostmt := range stmts {
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table == nil {
				continue
			}
			for _, c := range n.Constraints {
				if c == nil || !omniConstraintIsIndexFamily(c.Type) {
					continue
				}
				if dup, found := omniHasDuplicateString(omniConstraintColumnNames(c)); found {
					hits = append(hits, hit{
						tp:     omniIndexTypeString(c.Type),
						table:  n.Table.Name,
						index:  omniConstraintAdviceName(c),
						column: dup,
						line:   ostmt.AbsoluteLine(c.Loc.Start),
					})
				}
			}
		case *ast.CreateIndexStmt:
			if n.Table == nil {
				continue
			}
			if dup, found := omniHasDuplicateString(omniIndexColumns(n.Columns)); found {
				hits = append(hits, hit{
					tp:     "INDEX",
					table:  n.Table.Name,
					index:  n.IndexName,
					column: dup,
					line:   ostmt.FirstTokenLine(),
				})
			}
		case *ast.AlterTableStmt:
			if n.Table == nil {
				continue
			}
			stmtLine := ostmt.FirstTokenLine()
			for _, cmd := range n.Commands {
				if cmd == nil || cmd.Constraint == nil {
					continue
				}
				// Cumulative #17 sibling-parity: ATAddIndex paired
				// with ATAddConstraint even though tidb omni emits
				// only the latter today.
				if cmd.Type != ast.ATAddConstraint && cmd.Type != ast.ATAddIndex {
					continue
				}
				c := cmd.Constraint
				if !omniConstraintIsIndexFamily(c.Type) {
					continue
				}
				if dup, found := omniHasDuplicateString(omniConstraintColumnNames(c)); found {
					hits = append(hits, hit{
						tp:     omniIndexTypeString(c.Type),
						table:  n.Table.Name,
						index:  omniConstraintAdviceName(c),
						column: dup,
						line:   stmtLine,
					})
				}
			}
		default:
		}
	}

	adviceList := make([]*storepb.Advice, 0, len(hits))
	for _, h := range hits {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.DuplicateColumnInIndex.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("%s `%s` has duplicate column `%s`.`%s`", h.tp, h.index, h.table, h.column),
			StartPosition: common.ConvertANTLRLineToPosition(h.line),
		})
	}
	return adviceList, nil
}

// omniConstraintIsIndexFamily reports whether the constraint type
// participates in the duplicate-column check: PRIMARY KEY, UNIQUE,
// INDEX/KEY, or FOREIGN KEY. Omni's `ConstrUnique` unifies the 3
// pingcap UNIQUE syntactic forms (parser produces ConstraintUniq=4
// for bare UNIQUE / UNIQUE KEY / UNIQUE INDEX; UniqKey=5 / UniqIndex=6
// are defined enum values but parser-unreachable — see top-level
// docstring's cumulative #2 note). Mechanical port.
func omniConstraintIsIndexFamily(t ast.ConstraintType) bool {
	switch t {
	case ast.ConstrPrimaryKey, ast.ConstrUnique, ast.ConstrIndex, ast.ConstrForeignKey:
		return true
	default:
		return false
	}
}

// omniIndexTypeString returns the display string for the given
// constraint type, used in the duplicate-column advice content.
// Mirrors pre-omni `indexTypeString`. Pre-omni had 3 separate UNIQUE
// cases all mapping to "UNIQUE KEY" — omni's unified ConstrUnique
// renders identically.
func omniIndexTypeString(t ast.ConstraintType) string {
	switch t {
	case ast.ConstrPrimaryKey:
		return "PRIMARY KEY"
	case ast.ConstrUnique:
		return "UNIQUE KEY"
	case ast.ConstrForeignKey:
		return "FOREIGN KEY"
	case ast.ConstrIndex:
		return "INDEX"
	default:
		return "INDEX"
	}
}

// omniConstraintColumnNames extracts the plain-column names for the
// duplicate-column check. Omni splits the storage by constraint type:
//   - INDEX / PK / UNIQUE store keys in `IndexColumns []*IndexColumn`;
//     each may carry an `Expr` that's either `*ColumnRef` (plain
//     column) or another expression (functional index). The pre-omni
//     filter `if key.Expr == nil` maps to "skip non-ColumnRef Exprs"
//     in omni; we reuse the existing `omniIndexColumns` helper which
//     applies that filter.
//   - FOREIGN KEY stores its local columns in `Columns []string`
//     (verified empirically against omni parser source — FK
//     `constr.Columns = cols` is the only population path; IndexColumns
//     stays nil). Return those directly.
func omniConstraintColumnNames(c *ast.Constraint) []string {
	if c == nil {
		return nil
	}
	if c.Type == ast.ConstrForeignKey {
		return c.Columns
	}
	return omniIndexColumns(c.IndexColumns)
}

// omniHasDuplicateString returns the first repeating name in the
// slice, or "" / false if no duplicates. Used by
// index_no_duplicate_column to detect repeated column refs in index
// key lists after omniConstraintColumnNames normalization.
func omniHasDuplicateString(names []string) (string, bool) {
	seen := make(map[string]bool)
	for _, name := range names {
		if seen[name] {
			return name, true
		}
		seen[name] = true
	}
	return "", false
}
