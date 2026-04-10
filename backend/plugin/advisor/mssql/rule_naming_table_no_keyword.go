package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*NamingTableNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD, &NamingTableNoKeywordAdvisor{})
}

// NamingTableNoKeywordAdvisor is the advisor checking for table naming convention without keyword.
type NamingTableNoKeywordAdvisor struct {
}

// Check checks for table naming convention without keyword.
func (*NamingTableNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &namingTableNoKeywordRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type namingTableNoKeywordRule struct {
	OmniBaseRule
}

func (*namingTableNoKeywordRule) Name() string {
	return "NamingTableNoKeywordRule"
}

func (r *namingTableNoKeywordRule) OnStatement(node ast.Node) {
	ct, ok := node.(*ast.CreateTableStmt)
	if !ok || ct.Name == nil {
		return
	}
	tableName := ct.Name.Object
	if tableName == "" {
		return
	}
	if tsqlparser.IsTSQLReservedKeyword(tableName, false) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NameIsKeywordIdentifier.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Table name [%s] is a reserved keyword and should be avoided.", tableName),
			StartPosition: &storepb.Position{Line: r.LocToLine(ct.Loc)},
		})
	}
}
