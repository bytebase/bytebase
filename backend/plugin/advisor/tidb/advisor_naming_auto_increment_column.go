package tidb

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingAutoIncrementColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, &NamingAutoIncrementColumnAdvisor{})
}

// NamingAutoIncrementColumnAdvisor is the advisor checking for auto-increment naming convention.
type NamingAutoIncrementColumnAdvisor struct {
}

// Check checks for auto-increment naming convention.
func (*NamingAutoIncrementColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for this rule")
	}
	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}
	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		cols := collectColumnViolations(ostmt, func(col *ast.ColumnDef) bool {
			return col != nil && col.AutoIncrement
		})
		// Each AUTO_INCREMENT column may produce TWO advices (format
		// mismatch + length overflow). Pingcap-typed predecessor emitted
		// both independently per column — preserve.
		for _, col := range cols {
			if !format.MatchString(col.column) {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.NamingAutoIncrementColumnConventionMismatch.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, naming format should be %q", col.table, col.column, format),
					StartPosition: common.ConvertANTLRLineToPosition(col.line),
				})
			}
			if maxLength > 0 && len(col.column) > maxLength {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.NamingAutoIncrementColumnConventionMismatch.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, its length should be within %d characters", col.table, col.column, maxLength),
					StartPosition: common.ConvertANTLRLineToPosition(col.line),
				})
			}
		}
	}

	return adviceList, nil
}
