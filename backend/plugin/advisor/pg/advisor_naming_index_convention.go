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
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_IDX, &NamingIndexConventionAdvisor{})
}

// NamingIndexConventionAdvisor is the advisor checking for index naming
// convention. The judgment lives in omni (pg/v1review.NamingIndex); this
// adapter validates the template's tokens, converts the rule's payload into
// the omni config, and renders the findings as advice.
type NamingIndexConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingIndexConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	schema := originalSchema(checkCtx.OriginalMetadata)

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
		// V1 anchored every finding on the statement text's first
		// non-whitespace line, which can be a leading comment above the
		// statement node the finding's range covers.
		line := (&BaseRule{StmtText: stmt.Text}).ContentStartLine() + int32(stmt.BaseLine())
		for _, f := range pgv1review.NamingIndex([]ast.Node{node}, cfg, schema) {
			advice := &storepb.Advice{
				Status:        level,
				Code:          code.NamingIndexConventionMismatch.Int32(),
				Title:         title,
				StartPosition: &storepb.Position{Line: line},
			}
			switch f.Code {
			case v1review.PatternInvalid:
				advice.Code = code.Internal.Int32()
				advice.Title = "Internal error for index naming convention rule"
				advice.Content = fmt.Sprintf("Failed to compile regex: %s", f.Detail)
			case v1review.NameMismatch:
				advice.Content = fmt.Sprintf("Index in table %q mismatches the naming convention, expect %q but found %q", f.Table, f.Pattern, f.Name)
			case v1review.NameTooLong:
				advice.Content = fmt.Sprintf("Index %q in table %q mismatches the naming convention, its length should be within %d characters", f.Name, f.Table, maxLength)
			default:
				continue
			}
			adviceList = append(adviceList, advice)
		}
	}
	return adviceList, nil
}
