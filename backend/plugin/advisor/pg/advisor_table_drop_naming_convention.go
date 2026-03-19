package pg

import (
	"context"
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking for table drop with naming convention.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for table drop with naming convention.
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

	rule := &tableDropNamingConventionRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		format: format,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableDropNamingConventionRule struct {
	OmniBaseRule
	format *regexp.Regexp
}

// Name returns the rule name.
func (*tableDropNamingConventionRule) Name() string {
	return "table.drop-naming-convention"
}

// OnStatement is called for each top-level statement AST node.
func (r *tableDropNamingConventionRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.DropStmt)
	if !ok {
		return
	}
	if n.RemoveType != int(ast.OBJECT_TABLE) {
		return
	}

	for _, obj := range omniDropObjects(n) {
		tableName := obj[1] // [schema, name]
		if !r.format.MatchString(tableName) {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.TableDropNamingConventionMismatch.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", tableName, r.format),
				StartPosition: &storepb.Position{
					Line:   r.ContentStartLine(),
					Column: 0,
				},
			})
		}
	}
}
