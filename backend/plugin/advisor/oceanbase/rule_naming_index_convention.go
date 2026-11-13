package oceanbase

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_INDEX_IDX, &NamingIndexConventionAdvisor{})
}

// NamingIndexConventionAdvisor is the advisor checking for index naming
// convention. The judgment lives in omni (oceanbase/v1review.NamingIndex); this
// adapter checks the template's tokens, converts the rule's payload into the omni
// config, hands over the pre-change schema and renders the findings as advice.
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

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := oceanbaseparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range oceanbasev1review.NamingIndex([]ast.Node{node}, cfg, renameIndexSchema(node, checkCtx.OriginalMetadata)) {
			var content string
			switch f.Code {
			case v1review.PatternInvalid:
				adviceList = append(adviceList, &storepb.Advice{
					Status:  level,
					Code:    code.Internal.Int32(),
					Title:   "Internal error for unique key naming convention rule",
					Content: fmt.Sprintf("%q meet internal error %q", strings.TrimRight(strings.TrimSpace(stmt.Text), ";"), f.Detail),
				})
				continue
			case v1review.NameMismatch:
				content = fmt.Sprintf("Index in table `%s` mismatches the naming convention, expect %q but found `%s`", f.Table, f.Pattern, f.Name)
			case v1review.NameTooLong:
				content = fmt.Sprintf("Index `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", f.Name, f.Table, maxLength)
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NamingIndexConventionMismatch.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
			})
		}
	}
	return adviceList, nil
}

// renameIndexSchema returns the pre-change schema for a statement that renames
// an index, the only one whose judgment reads it. The metadata is not checked
// for nil: the V1 rules dereferenced it at each RENAME INDEX, so a nil
// OriginalMetadata panics on these statements alone and advisor.Check reports
// the panic as the rule's error.
func renameIndexSchema(node ast.Node, original *model.DatabaseMetadata) v1review.Schema {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return v1review.Schema{}
	}
	for _, cmd := range alter.Commands {
		if cmd != nil && cmd.Type == ast.ATRenameIndex {
			return v1review.Schema{Metadata: original.GetProto(), ObjectCaseSensitive: original.GetIsObjectCaseSensitive()}
		}
	}
	return v1review.Schema{}
}
