package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableDisallowSetCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_SET_CHARSET, &TableDisallowSetCharsetAdvisor{})
}

type TableDisallowSetCharsetAdvisor struct {
}

func (*TableDisallowSetCharsetAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableDisallowSetCharsetOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableDisallowSetCharsetOmniRule struct {
	OmniBaseRule
}

func (*tableDisallowSetCharsetOmniRule) Name() string {
	return "TableDisallowSetCharsetRule"
}

func (r *tableDisallowSetCharsetOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *tableDisallowSetCharsetOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	for _, opt := range n.Options {
		if opt != nil && strings.EqualFold(opt.Name, "CHARSET") || strings.EqualFold(opt.Name, "CHARACTER SET") || strings.EqualFold(opt.Name, "DEFAULT CHARSET") || strings.EqualFold(opt.Name, "DEFAULT CHARACTER SET") {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.DisallowSetCharset.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Set charset on tables is disallowed, but \"%s\" uses", r.QueryText()),
				StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
			})
		}
	}
}

func (r *tableDisallowSetCharsetOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		if cmd.Type == ast.ATConvertCharset {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.DisallowSetCharset.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Set charset on tables is disallowed, but \"%s\" uses", r.QueryText()),
				StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
			})
		}
	}
}
