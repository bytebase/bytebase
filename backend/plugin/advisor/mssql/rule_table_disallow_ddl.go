package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DDL, &TableDisallowDDLAdvisor{})
}

type TableDisallowDDLAdvisor struct{}

func (*TableDisallowDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for table disallow DDL rule")
	}

	disallowList := make([]string, len(stringArrayPayload.List))
	for i, s := range stringArrayPayload.List {
		disallowList[i] = strings.ToLower(s)
	}
	rule := &tableDisallowDDLRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		disallowList: disallowList,
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableDisallowDDLRule struct {
	OmniBaseRule
	disallowList []string
}

func (*tableDisallowDDLRule) Name() string {
	return "TableDisallowDDLRule"
}

func (r *tableDisallowDDLRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		if n.Name != nil {
			r.checkTableName(normalizeTableRef(n.Name, "", ""), r.LocToLine(n.Loc))
		}
	case *ast.AlterTableStmt:
		if n.Name != nil {
			r.checkTableName(normalizeTableRef(n.Name, "", ""), r.LocToLine(n.Loc))
		}
	case *ast.DropStmt:
		if n.ObjectType != ast.DropTable || n.Names == nil {
			return
		}
		for _, item := range n.Names.Items {
			ref, ok := item.(*ast.TableRef)
			if !ok || ref == nil {
				continue
			}
			r.checkTableName(normalizeTableRef(ref, "", ""), r.LocToLine(n.Loc))
		}
	case *ast.TruncateStmt:
		if n.Table != nil {
			r.checkTableName(normalizeTableRef(n.Table, "", ""), r.LocToLine(n.Loc))
		}
	default:
	}
}

func (r *tableDisallowDDLRule) checkTableName(normalizedTableName string, line int32) {
	for _, disallow := range r.disallowList {
		if normalizedTableName == disallow {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.TableDisallowDDL.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("DDL is disallowed on table %s.", normalizedTableName),
				StartPosition: &storepb.Position{Line: line},
			})
			return
		}
	}
}
