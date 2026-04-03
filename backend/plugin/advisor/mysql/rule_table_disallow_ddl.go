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
	_ advisor.Advisor = (*TableDisallowDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DDL, &TableDisallowDDLAdvisor{})
}

// TableDisallowDDLAdvisor is the advisor checking for disallow DDL on specific tables.
type TableDisallowDDLAdvisor struct {
}

func (*TableDisallowDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	rule := &tableDisallowDDLOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		disallowList: stringArrayPayload.List,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableDisallowDDLOmniRule struct {
	OmniBaseRule
	disallowList []string
}

func (*tableDisallowDDLOmniRule) Name() string {
	return "TableDisallowDDLRule"
}

func (r *tableDisallowDDLOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		if n.Table != nil {
			r.checkTableName(n.Table.Name, r.LocToLine(n.Loc))
		}
	case *ast.AlterTableStmt:
		if n.Table != nil {
			r.checkTableName(n.Table.Name, r.LocToLine(n.Loc))
		}
	case *ast.DropTableStmt:
		for _, tbl := range n.Tables {
			r.checkTableName(tbl.Name, r.LocToLine(n.Loc))
		}
	case *ast.RenameTableStmt:
		for _, pair := range n.Pairs {
			if pair.Old != nil {
				r.checkTableName(pair.Old.Name, r.LocToLine(n.Loc))
			}
		}
	case *ast.TruncateStmt:
		for _, tbl := range n.Tables {
			r.checkTableName(tbl.Name, r.LocToLine(n.Loc))
		}
	default:
	}
}

func (r *tableDisallowDDLOmniRule) checkTableName(tableName string, lineNumber int32) {
	for _, disallow := range r.disallowList {
		if tableName == disallow {
			absoluteLine := r.BaseLine + int(lineNumber)
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.TableDisallowDDL.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("DDL is disallowed on table %s.", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(absoluteLine),
			})
			return
		}
	}
}
