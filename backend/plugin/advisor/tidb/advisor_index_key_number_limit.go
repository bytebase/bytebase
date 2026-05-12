package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*IndexKeyNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, &IndexKeyNumberLimitAdvisor{})
}

// IndexKeyNumberLimitAdvisor flags index/constraint declarations whose
// number of key columns exceeds the configured maximum.
type IndexKeyNumberLimitAdvisor struct{}

// Check fires on per-constraint key counts in CREATE TABLE, CREATE
// INDEX, and ALTER TABLE ADD CONSTRAINT. No cross-stmt state. Recipe A.
//
// Cumulative #2 coverage: pingcap-tidb's pre-omni indexKeyNumber()
// helper handled `ConstraintUniq`, `ConstraintUniqKey`, and
// `ConstraintUniqIndex` as three distinct enum values. Omni unifies
// all three under `ConstrUnique` (verified empirically: parsing
// `UNIQUE(a)`, `UNIQUE KEY uk(a)`, `UNIQUE INDEX ui(a)` all yields
// `Type=ConstrUnique`). The omni port matches the single arm and
// covers all three forms mechanically — NOT a regression.
//
// Cumulative #19 (case-sensitivity): pre-omni used `.O` throughout
// (no `.L`). Omni preserves user case via direct strings. Mechanical.
func (*IndexKeyNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		return nil, errors.New("number_payload is required for index key number limit rule")
	}
	maximum := int(numberPayload.Number)
	if maximum <= 0 {
		return nil, nil
	}
	title := checkCtx.Rule.Type.String()

	type violation struct {
		table string
		index string
		line  int
	}
	var hits []violation

	for _, ostmt := range stmts {
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table == nil {
				continue
			}
			for _, c := range n.Constraints {
				if c == nil {
					continue
				}
				if omniIndexKeyCount(c) > maximum {
					hits = append(hits, violation{
						table: n.Table.Name,
						index: omniConstraintAdviceName(c),
						line:  ostmt.AbsoluteLine(c.Loc.Start),
					})
				}
			}
		case *ast.CreateIndexStmt:
			if n.Table == nil {
				continue
			}
			if len(n.Columns) > maximum {
				hits = append(hits, violation{
					table: n.Table.Name,
					index: n.IndexName,
					line:  ostmt.FirstTokenLine(),
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
				// Cumulative #17 sibling-parity convention: tidb omni
				// emits only ATAddConstraint for all `ALTER TABLE ADD
				// ...` forms today, but the dual arm is the recommended
				// convention (batch 4 naming-trio + batch 8 index spine
				// + utils.go collectIndexFamilyAlterTable) for forward-
				// compat against grammar evolution that may start
				// emitting ATAddIndex.
				if cmd.Type != ast.ATAddConstraint && cmd.Type != ast.ATAddIndex {
					continue
				}
				if omniIndexKeyCount(cmd.Constraint) > maximum {
					hits = append(hits, violation{
						table: n.Table.Name,
						index: omniConstraintAdviceName(cmd.Constraint),
						line:  stmtLine,
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
			Code:          code.IndexKeyNumberExceedsLimit.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("The number of index `%s` in table `%s` should be not greater than %d", h.index, h.table, maximum),
			StartPosition: common.ConvertANTLRLineToPosition(h.line),
		})
	}
	return adviceList, nil
}

// omniIndexKeyCount returns the number of key columns declared by the
// given constraint. INDEX/PK/UNIQUE store keys in `IndexColumns`;
// FOREIGN KEY stores its local columns in `Columns []string` with
// `IndexColumns` empty (verified empirically).
func omniIndexKeyCount(c *ast.Constraint) int {
	if c == nil {
		return 0
	}
	switch c.Type {
	case ast.ConstrIndex, ast.ConstrPrimaryKey, ast.ConstrUnique:
		return len(c.IndexColumns)
	case ast.ConstrForeignKey:
		return len(c.Columns)
	default:
		return 0
	}
}

// omniConstraintAdviceName returns the constraint name suitable for
// embedding in advice content. Falls back to "PRIMARY" for unnamed
// PRIMARY KEY constraints (cumulative #28: pingcap-tidb accepted the
// non-standard `PRIMARY KEY index_name (cols)` extension and captured
// the index_name; omni follows standard MySQL grammar where PRIMARY
// KEY doesn't accept an index_name and silently drops it. "PRIMARY"
// is MySQL's canonical internal name for the primary key — better UX
// than the empty backticks the raw `c.Name` would produce).
func omniConstraintAdviceName(c *ast.Constraint) string {
	if c == nil {
		return ""
	}
	if c.Name != "" {
		return c.Name
	}
	if c.Type == ast.ConstrPrimaryKey {
		return "PRIMARY"
	}
	return ""
}
