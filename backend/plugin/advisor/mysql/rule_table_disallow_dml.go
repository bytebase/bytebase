package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableDisallowDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DML, &TableDisallowDMLAdvisor{})
}

// TableDisallowDMLAdvisor is the advisor checking for disallow DML on specific tables.
type TableDisallowDMLAdvisor struct {
}

func (*TableDisallowDMLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	rule := &tableDisallowDMLOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		disallowList: stringArrayPayload.List,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableDisallowDMLOmniRule struct {
	OmniBaseRule
	disallowList []string
}

func (*tableDisallowDMLOmniRule) Name() string {
	return "TableDisallowDMLRule"
}

func (r *tableDisallowDMLOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.DeleteStmt:
		for _, name := range omniTableExprNames(n.Tables) {
			r.checkTableName(name, r.LocToLine(n.Loc))
		}
	case *ast.InsertStmt:
		if n.Table != nil {
			r.checkTableName(n.Table.Name, r.LocToLine(n.Loc))
		}
	case *ast.SelectStmt:
		if n.Into != nil && n.Into.Outfile != "" {
			r.checkTableName(n.Into.Outfile, r.LocToLine(n.Loc))
		}
	case *ast.UpdateStmt:
		for _, name := range omniTableExprNames(n.Tables) {
			r.checkTableName(name, r.LocToLine(n.Loc))
		}
	default:
	}
}

func (r *tableDisallowDMLOmniRule) checkTableName(tableName string, lineNumber int32) {
	for _, disallow := range r.disallowList {
		if tableName == disallow {
			absoluteLine := r.BaseLine + int(lineNumber)
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.TableDisallowDML.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("DML is disallowed on table %s.", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(absoluteLine),
			})
			return
		}
	}
}

// omniTableExprNames extracts table names from a slice of TableExpr.
func omniTableExprNames(exprs []ast.TableExpr) []string {
	var names []string
	for _, expr := range exprs {
		ast.Inspect(expr, func(n ast.Node) bool {
			t, ok := n.(*ast.TableRef)
			if !ok {
				return true
			}
			names = append(names, t.Name)
			return false
		})
	}
	return names
}
