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
	_ advisor.Advisor = (*ColumnAutoIncrementMustUnsignedAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, &ColumnAutoIncrementMustUnsignedAdvisor{})
}

// ColumnAutoIncrementMustUnsignedAdvisor is the advisor checking for unsigned auto-increment column.
type ColumnAutoIncrementMustUnsignedAdvisor struct {
}

// Check checks for unsigned auto-increment column.
func (*ColumnAutoIncrementMustUnsignedAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnAutoIncrementMustUnsignedChecker{
		level: level,
		title: checkCtx.Rule.Type.String(),
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type columnAutoIncrementMustUnsignedChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

func (c *columnAutoIncrementMustUnsignedChecker) checkStmt(ostmt OmniStmt) {
	cols := collectColumnViolations(ostmt, func(col *ast.ColumnDef) bool {
		return !autoIncrementColumnIsUnsignedOmni(col)
	})
	for _, col := range cols {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.AutoIncrementColumnSigned.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("Auto-increment column `%s`.`%s` is not UNSIGNED type", col.table, col.column),
			StartPosition: common.ConvertANTLRLineToPosition(col.line),
		})
	}
}

// autoIncrementColumnIsUnsignedOmni returns true unless the column is
// AUTO_INCREMENT and neither Unsigned nor Zerofill is set. Pingcap-typed
// `mysql.HasUnsignedFlag(column.Tp.GetFlag())` returned true when EITHER
// the unsigned flag OR the zerofill flag was set (ZEROFILL implies
// UNSIGNED in MySQL). Omni splits the bits: `col.TypeName.Unsigned` and
// `col.TypeName.Zerofill` are separate booleans. Check both to preserve
// pingcap-tidb behavior. Mirrors mysql analog at
// `mysql/rule_column_auto_increment_must_unsigned.go:102`.
func autoIncrementColumnIsUnsignedOmni(col *ast.ColumnDef) bool {
	if col == nil {
		return true
	}
	if !col.AutoIncrement {
		return true
	}
	if col.TypeName == nil {
		return false
	}
	return col.TypeName.Unsigned || col.TypeName.Zerofill
}
