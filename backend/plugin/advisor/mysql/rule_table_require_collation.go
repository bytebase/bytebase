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
	_ advisor.Advisor = (*TableRequireCollationAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_REQUIRE_COLLATION, &TableRequireCollationAdvisor{})
}

// TableRequireCollationAdvisor is the advisor checking for require collation.
type TableRequireCollationAdvisor struct {
}

func (*TableRequireCollationAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableRequireCollationOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableRequireCollationOmniRule struct {
	OmniBaseRule
}

func (*tableRequireCollationOmniRule) Name() string {
	return "TableRequireCollationRule"
}

func (r *tableRequireCollationOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.CreateTableStmt)
	if !ok {
		return
	}
	r.checkCreateTable(n)
}

func (r *tableRequireCollationOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	if tableName == "" {
		return
	}

	hasCollation := false
	for _, opt := range n.Options {
		if opt != nil && (strings.EqualFold(opt.Name, "COLLATE") || strings.EqualFold(opt.Name, "DEFAULT COLLATE")) {
			hasCollation = true
			break
		}
	}
	if !hasCollation {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NoCollation.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Table %s does not have a collation specified", tableName),
			StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
		})
	}
}
