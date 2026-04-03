package mysql

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bytebase/omni/mysql/ast"
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
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, &TableDropNamingConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, &TableDropNamingConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking the MySQLTableDropNamingConvention rule.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for drop table naming convention.
func (*TableDropNamingConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := &tableDropNamingConventionOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		format: format,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableDropNamingConventionOmniRule struct {
	OmniBaseRule
	format *regexp.Regexp
}

func (*tableDropNamingConventionOmniRule) Name() string {
	return "TableDropNamingConventionRule"
}

func (r *tableDropNamingConventionOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.DropTableStmt)
	if !ok {
		return
	}
	for _, tbl := range n.Tables {
		if !r.format.MatchString(tbl.Name) {
			absoluteLine := r.BaseLine + int(r.LocToLine(n.Loc))
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.TableDropNamingConventionMismatch.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", tbl.Name, r.format),
				StartPosition: common.ConvertANTLRLineToPosition(absoluteLine),
			})
		}
	}
}
