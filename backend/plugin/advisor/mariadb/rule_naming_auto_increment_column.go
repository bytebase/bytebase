package mariadb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

var (
	_ advisor.Advisor = (*NamingAutoIncrementColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, &NamingAutoIncrementColumnAdvisor{})
}

// NamingAutoIncrementColumnAdvisor is the advisor checking for auto-increment
// naming convention. The judgment lives in omni
// (mariadb/v1review.NamingAutoIncrementColumn); this adapter converts the
// rule's payload into the omni config and renders the findings as advice.
type NamingAutoIncrementColumnAdvisor struct {
}

// Check checks for auto-increment naming convention.
func (*NamingAutoIncrementColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mariadbparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		findings, err := mariadbv1review.NamingAutoIncrementColumn([]ast.Node{node}, cfg)
		if err != nil {
			return nil, err
		}
		for _, f := range findings {
			var content string
			switch f.Code {
			case v1review.NameMismatch:
				content = fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, naming format should be %q", f.Table, f.Name, namingPayload.Format)
			case v1review.NameTooLong:
				content = fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, its length should be within %d characters", f.Table, f.Name, maxLength)
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NamingAutoIncrementColumnConventionMismatch.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
			})
		}
	}
	return adviceList, nil
}
