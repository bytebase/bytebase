package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_COLUMN, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming
// convention. The judgment lives in omni (pg/v1review.NamingColumn); this
// adapter converts the rule's payload into the omni config and renders the
// findings as advice.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		findings, err := pgv1review.NamingColumn([]ast.Node{node}, cfg)
		if err != nil {
			return nil, err
		}
		for _, f := range findings {
			var content string
			switch f.Code {
			case v1review.NameMismatch:
				content = fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, naming format should be %q", f.Table, f.Name, namingPayload.Format)
			case v1review.NameTooLong:
				content = fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, its length should be within %d characters", f.Table, f.Name, maxLength)
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Code:    code.NamingColumnConventionMismatch.Int32(),
				Title:   title,
				Content: content,
				// V1 reports the line of the name's first occurrence in the
				// statement text, which is not always the column itself.
				StartPosition: &storepb.Position{
					Line: (&BaseRule{StmtText: stmt.Text}).FindLineByName(f.Name) + int32(stmt.BaseLine()),
				},
			})
		}
	}
	return adviceList, nil
}
