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
	_ advisor.Advisor = (*InsertMustSpecifyColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, &InsertMustSpecifyColumnAdvisor{})
}

// InsertMustSpecifyColumnAdvisor is the advisor checking for to enforce column specified.
type InsertMustSpecifyColumnAdvisor struct {
}

// Check checks for to enforce column specified.
func (*InsertMustSpecifyColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &insertMustSpecifyColumnChecker{
		level: level,
		title: checkCtx.Rule.Type.String(),
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type insertMustSpecifyColumnChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

func (c *insertMustSpecifyColumnChecker) checkStmt(ostmt OmniStmt) {
	node, ok := ostmt.Node.(*ast.InsertStmt)
	if !ok {
		return
	}
	// Already has an explicit column list — no advice.
	if len(node.Columns) > 0 {
		return
	}
	// Mirror mysql/rule_insert_must_specify_column.go: flag only the VALUES
	// and SELECT forms when they lack an explicit column list. INSERT ... SET
	// (and equivalent REPLACE ... SET, TABLE-source, DEFAULT VALUES, etc.)
	// specify their target columns through assignments or source semantics
	// and were not flagged by the prior pingcap-AST rule. Without this carve-
	// out the omni-migrated rule false-positives on every SET-form insert.
	isValuesForm := len(node.Values) > 0
	isSelectForm := node.Select != nil
	if !isValuesForm && !isSelectForm {
		return
	}
	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:        c.level,
		Code:          code.InsertNotSpecifyColumn.Int32(),
		Title:         c.title,
		Content:       fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", ostmt.Text),
		StartPosition: common.ConvertANTLRLineToPosition(ostmt.AbsoluteLine(node.Loc.Start)),
	})
}
