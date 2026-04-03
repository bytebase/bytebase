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
	_ advisor.Advisor = (*TableRequireCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_REQUIRE_CHARSET, &TableRequireCharsetAdvisor{})
}

// TableRequireCharsetAdvisor is the advisor checking for require charset.
type TableRequireCharsetAdvisor struct {
}

func (*TableRequireCharsetAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableRequireCharsetOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableRequireCharsetOmniRule struct {
	OmniBaseRule
}

func (*tableRequireCharsetOmniRule) Name() string {
	return "TableRequireCharsetRule"
}

func (r *tableRequireCharsetOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.CreateTableStmt)
	if !ok {
		return
	}
	r.checkCreateTable(n)
}

func (r *tableRequireCharsetOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	if tableName == "" {
		return
	}

	hasCharset := false
	for _, opt := range n.Options {
		if opt != nil && (strings.EqualFold(opt.Name, "CHARSET") || strings.EqualFold(opt.Name, "CHARACTER SET") || strings.EqualFold(opt.Name, "DEFAULT CHARSET") || strings.EqualFold(opt.Name, "DEFAULT CHARACTER SET")) {
			hasCharset = true
			break
		}
	}
	if !hasCharset {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NoCharset.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Table %s does not have a character set specified", tableName),
			StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
		})
	}
}
