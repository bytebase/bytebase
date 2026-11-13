package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_INDEX_IDX, &NamingIndexConventionAdvisor{})
}

// NamingIndexConventionAdvisor is the advisor checking for index naming
// convention. The judgment lives in omni (mysql/v1review.NamingIndex); this
// adapter validates the template's tokens, converts the rule's payload into
// the omni config and renders the findings as advice.
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
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range mysqlv1review.NamingIndex([]ast.Node{node}, cfg, originalSchema(checkCtx.OriginalMetadata, node)) {
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

// originalSchema returns the pre-change schema an index naming rule judges
// one statement against: the metadata for a statement that renames an index,
// and nothing for any other, which reads none. The metadata is read
// unguarded, so nil metadata panics on a RENAME INDEX only, and advisor.Check
// reports the panic as the rule's error.
func originalSchema(metadata *model.DatabaseMetadata, node ast.Node) v1review.Schema {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return v1review.Schema{}
	}
	for _, cmd := range alter.Commands {
		if cmd != nil && cmd.Type == ast.ATRenameIndex {
			return v1review.Schema{Metadata: metadata.GetProto(), ObjectCaseSensitive: metadata.GetIsObjectCaseSensitive()}
		}
	}
	return v1review.Schema{}
}
