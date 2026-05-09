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
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking the MySQLTableDropNamingConvention rule.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for drop table naming convention.
func (*TableDropNamingConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		return nil, errors.New("naming_payload is required for table drop naming convention rule")
	}

	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	checker := &namingDropTableConventionChecker{
		level:  level,
		title:  checkCtx.Rule.Type.String(),
		format: format,
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type namingDropTableConventionChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
}

func (c *namingDropTableConventionChecker) checkStmt(ostmt OmniStmt) {
	node, ok := ostmt.Node.(*ast.DropTableStmt)
	if !ok {
		return
	}
	line := ostmt.AbsoluteLine(node.Loc.Start)
	for _, table := range node.Tables {
		if table == nil {
			continue
		}
		if !c.format.MatchString(table.Name) {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.TableDropNamingConventionMismatch.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", table.Name, c.format),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
		}
	}
}
