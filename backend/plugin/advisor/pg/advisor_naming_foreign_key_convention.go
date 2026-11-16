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
	_ advisor.Advisor = (*NamingFKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_FK, &NamingFKConventionAdvisor{})
}

// NamingFKConventionAdvisor is the advisor checking for foreign key naming
// convention. The judgment lives in omni (pg/v1review.NamingForeignKey);
// this adapter validates the template's tokens, converts the rule's payload
// into the omni config, and renders the findings as advice.
type NamingFKConventionAdvisor struct {
}

// Check checks for foreign key naming convention.
func (*NamingFKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for this rule")
	}

	format := namingPayload.Format
	templateList, _ := advisor.ParseTemplateTokens(format)

	for _, key := range templateList {
		if _, ok := advisor.TemplateNamingTokens[checkCtx.Rule.Type][key]; !ok {
			return nil, errors.Errorf("invalid template %s for rule %s", key, checkCtx.Rule.Type)
		}
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}
	cfg := v1review.TemplateConfig{Format: format, MaxLength: maxLength}

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
		findings, err := pgv1review.NamingForeignKey([]ast.Node{node}, cfg)
		if err != nil {
			return nil, err
		}
		for _, f := range findings {
			advice := &storepb.Advice{
				Status: level,
				Code:   code.NamingFKConventionMismatch.Int32(),
				Title:  title,
				// V1 reports the line of the name's first occurrence in the
				// statement text, which is not always the constraint itself.
				StartPosition: &storepb.Position{
					Line: (&BaseRule{StmtText: stmt.Text}).FindLineByName(f.Name) + int32(stmt.BaseLine()),
				},
			}
			switch f.Code {
			case v1review.PatternInvalid:
				advice.Code = code.Internal.Int32()
				advice.Title = "Internal error for foreign key naming convention rule"
				advice.Content = fmt.Sprintf("Failed to compile regex: %s", f.Detail)
			case v1review.NameMismatch:
				advice.Content = fmt.Sprintf(`Foreign key in table "%s" mismatches the naming convention, expect %q but found "%s"`, f.Table, f.Pattern, f.Name)
			case v1review.NameTooLong:
				advice.Content = fmt.Sprintf(`Foreign key "%s" in table "%s" mismatches the naming convention, its length should be within %d characters`, f.Name, f.Table, maxLength)
			default:
				continue
			}
			adviceList = append(adviceList, advice)
		}
	}
	return adviceList, nil
}
