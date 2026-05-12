package tidb

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnAutoIncrementInitialValueAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, &ColumnAutoIncrementInitialValueAdvisor{})
}

// ColumnAutoIncrementInitialValueAdvisor is the advisor checking for auto-increment column initial value.
type ColumnAutoIncrementInitialValueAdvisor struct {
}

// Check checks for auto-increment column initial value.
func (*ColumnAutoIncrementInitialValueAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		return nil, errors.New("number_payload is required for column auto increment initial value rule")
	}
	checker := &columnAutoIncrementInitialValueChecker{
		level: level,
		title: checkCtx.Rule.Type.String(),
		value: int(numberPayload.Number),
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type columnAutoIncrementInitialValueChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	value      int
}

// checkStmt: only CREATE TABLE is in scope, matching pingcap-typed
// columnAutoIncrementInitialValueChecker.Enter (which only matched
// `*ast.CreateTableStmt`). The mysql analog ALSO handles
// `ALTER TABLE ... AUTO_INCREMENT = N` via the ATTableOption arm —
// that broader scope is a mysql-side behavior, NOT something the
// tidb migration should adopt (preserve pingcap behavior per
// invariant #7's caveat on mysql analog non-authority).
func (c *columnAutoIncrementInitialValueChecker) checkStmt(ostmt OmniStmt) {
	create, ok := ostmt.Node.(*ast.CreateTableStmt)
	if !ok || create.Table == nil {
		return
	}
	tableName := create.Table.Name
	stmtLine := ostmt.AbsoluteLine(create.Loc.Start)
	for _, opt := range create.Options {
		if opt == nil {
			continue
		}
		if !strings.EqualFold(opt.Name, "AUTO_INCREMENT") {
			continue
		}
		// omni stores the option value as a string; pingcap exposed it
		// as `option.UintValue` (uint64) and skipped the unparseable
		// case implicitly. Skip on parse failure here too.
		value, err := strconv.ParseUint(opt.Value, 10, 0)
		if err != nil {
			continue
		}
		if value != uint64(c.value) {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.AutoIncrementColumnInitialValueNotMatch.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("The initial auto-increment value in table `%s` is %v, which doesn't equal %v", tableName, value, c.value),
				StartPosition: common.ConvertANTLRLineToPosition(stmtLine),
			})
		}
	}
}
