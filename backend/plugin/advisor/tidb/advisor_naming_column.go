package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_COLUMN, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming
// convention. The judgment lives in omni (tidb/v1review.NamingColumn); this
// adapter converts the rule's payload into the omni config and renders the
// findings as advice.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
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

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}
	cfg := v1review.NamingConfig{Pattern: namingPayload.Format, MaxLength: maxLength}
	if _, err := cfg.Compile(); err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		findings, err := tidbv1review.NamingColumn([]ast.Node{stmt.Node}, cfg)
		if err != nil {
			return nil, err
		}
		for _, f := range findings {
			var content string
			switch f.Code {
			case v1review.NameMismatch:
				content = fmt.Sprintf("`%s`.`%s` mismatches column naming convention, naming format should be %q", f.Table, f.Name, namingPayload.Format)
			case v1review.NameTooLong:
				content = fmt.Sprintf("`%s`.`%s` mismatches column naming convention, its length should be within %d characters", f.Table, f.Name, maxLength)
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NamingColumnConventionMismatch.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: base.ConvertANTLRLineToPosition(stmt.AbsoluteLine(f.Start)),
			})
		}
	}
	return adviceList, nil
}
