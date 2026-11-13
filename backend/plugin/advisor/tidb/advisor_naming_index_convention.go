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
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_INDEX_IDX, &NamingIndexConventionAdvisor{})
}

// NamingIndexConventionAdvisor is the advisor checking for index naming
// convention. The judgment lives in omni (tidb/v1review.NamingIndex); this
// adapter validates the template's tokens, converts the rule's payload into
// the omni config, and renders the findings as advice.
type NamingIndexConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingIndexConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	templateList, _ := advisor.ParseTemplateTokens(namingPayload.Format)
	for _, key := range templateList {
		if _, ok := advisor.TemplateNamingTokens[checkCtx.Rule.Type][key]; !ok {
			return nil, errors.Errorf("invalid template %s for rule %s", key, checkCtx.Rule.Type)
		}
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}
	cfg := v1review.TemplateConfig{Format: namingPayload.Format, MaxLength: maxLength}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		findings := tidbv1review.NamingIndex([]ast.Node{stmt.Node}, cfg, renameIndexSchema(stmt.Node, checkCtx.OriginalMetadata))
		for _, f := range findings {
			var content string
			switch f.Code {
			case v1review.PatternInvalid:
				adviceList = append(adviceList, &storepb.Advice{
					Status:  level,
					Code:    code.Internal.Int32(),
					Title:   "Internal error for index naming convention rule",
					Content: fmt.Sprintf("%q meet internal error %q", stmt.TrimmedText(), f.Detail),
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
				StartPosition: base.ConvertANTLRLineToPosition(keyFindingLine(stmt, f)),
			})
		}
	}
	return adviceList, nil
}

// renameIndexSchema returns the pre-change schema for the index and unique
// key naming rules, read only for a statement that renames an index. Reading
// it lazily keeps V1's nil-metadata behavior: such a statement panics, which
// advisor.Check reports as the rule's error, and a review without one is
// judged in full.
func renameIndexSchema(node ast.Node, metadata *model.DatabaseMetadata) v1review.Schema {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok || alter.Table == nil {
		return v1review.Schema{}
	}
	for _, cmd := range alter.Commands {
		if cmd != nil && cmd.Type == ast.ATRenameIndex {
			return v1review.Schema{Metadata: metadata.GetProto(), ObjectCaseSensitive: metadata.GetIsObjectCaseSensitive()}
		}
	}
	return v1review.Schema{}
}

// keyFindingLine is the line of an index or unique key naming finding: the
// constraint's for a CREATE TABLE one, and the statement's first token for
// the others, which carry no range.
func keyFindingLine(stmt Stmt, f v1review.Finding) int {
	if f.Start != 0 || f.End != 0 {
		return stmt.AbsoluteLine(f.Start)
	}
	return stmt.FirstTokenLine()
}
