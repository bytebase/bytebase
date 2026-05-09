package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnDisallowSetCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, &ColumnDisallowSetCharsetAdvisor{})
}

// ColumnDisallowSetCharsetAdvisor checks for disallowed column-level
// CHARACTER SET clauses.
type ColumnDisallowSetCharsetAdvisor struct {
}

// Check is Recipe A. Same wrapper-safety rationale as the batch-6
// charset/collation/comment family.
//
// Cardinality: per-statement, single-advice with first-violation-wins.
// Pingcap's pattern set `code` and broke out of inner loops on first
// violation, with a single append after the visitor switch. Mirroring
// that semantic via early-exit `break` in the column loop / `done` flag
// for ALTER's outer spec loop.
//
// Closes BYT-9412 (getColumnCharset cleanup) by being the last consumer
// of the pingcap-typed getColumnCharset helper.
func (*ColumnDisallowSetCharsetAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	for _, ostmt := range stmts {
		var found bool
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			for _, col := range n.Columns {
				if col == nil {
					continue
				}
				if !columnCharsetAllowed(col) {
					found = true
					break // pingcap parity: first-violating-column wins
				}
			}
		case *ast.AlterTableStmt:
			for _, cmd := range n.Commands {
				if cmd == nil {
					continue
				}
				switch cmd.Type {
				case ast.ATAddColumn:
					for _, col := range addColumnTargets(cmd) {
						if col == nil {
							continue
						}
						if !columnCharsetAllowed(col) {
							found = true
							break
						}
					}
				case ast.ATChangeColumn, ast.ATModifyColumn:
					if cmd.Column != nil && !columnCharsetAllowed(cmd.Column) {
						found = true
					}
				default:
				}
				if found {
					// pingcap parity: outer specs loop breaks on first
					// violating spec.
					break
				}
			}
		default:
		}

		if !found {
			continue
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          advisorcode.SetColumnCharset.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Disallow set column charset but \"%s\" does", ostmt.TrimmedText()),
			StartPosition: common.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
		})
	}

	return adviceList, nil
}

// columnCharsetAllowed reports whether a column's CHARACTER SET is
// acceptable to this rule. Mirrors pingcap's checkCharset: empty (no
// clause) and "binary" (used for JSON-like types) are allowed; everything
// else is a violation.
func columnCharsetAllowed(col *ast.ColumnDef) bool {
	cs := omniColumnCharset(col)
	switch cs {
	case "", "binary":
		return true
	default:
		return false
	}
}
