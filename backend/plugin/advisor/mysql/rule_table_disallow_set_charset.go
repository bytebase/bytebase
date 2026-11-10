package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

	rule := &tableDisallowSetCharsetRule{
		BaseRule: BaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunRules(checkCtx.ParsedStatements, []Rule{rule}), nil
}

type tableDisallowSetCharsetRule struct {
	BaseRule
}

func (*tableDisallowSetCharsetRule) Name() string {
	return "TableDisallowSetCharsetRule"
}

func (r *tableDisallowSetCharsetRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *tableDisallowSetCharsetRule) checkCreateTable(n *ast.CreateTableStmt) {
	for _, opt := range n.Options {
		if opt != nil && strings.EqualFold(opt.Name, "CHARSET") || strings.EqualFold(opt.Name, "CHARACTER SET") || strings.EqualFold(opt.Name, "DEFAULT CHARSET") || strings.EqualFold(opt.Name, "DEFAULT CHARACTER SET") {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.DisallowSetCharset.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Set charset on tables is disallowed, but \"%s\" uses", r.QueryText()),
				StartPosition: base.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
			})
		}
	}
}

func (r *tableDisallowSetCharsetRule) checkAlterTable(n *ast.AlterTableStmt) {
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
				StartPosition: base.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
			})
		}
	}
}
