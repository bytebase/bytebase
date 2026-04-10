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

var (
	_ advisor.Advisor = (*TableDisallowDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DML, &TableDisallowDMLAdvisor{})
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
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for table disallow DML rule")
	}

	disallowList := make([]string, len(stringArrayPayload.List))
	for i, s := range stringArrayPayload.List {
		disallowList[i] = strings.ToLower(s)
	}
	rule := &tableDisallowDMLRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		disallowList: disallowList,
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableDisallowDMLRule struct {
	OmniBaseRule
	disallowList []string
}

func (*tableDisallowDMLRule) Name() string {
	return "TableDisallowDMLRule"
}

func (r *tableDisallowDMLRule) OnStatement(node ast.Node) {
	var tableName string
	var loc ast.Loc

	switch n := node.(type) {
	case *ast.InsertStmt:
		if n.Relation != nil {
			tableName = normalizeTableRef(n.Relation, "", "")
			loc = n.Loc
		}
	case *ast.UpdateStmt:
		if n.Relation != nil {
			tableName = normalizeTableRef(n.Relation, "", "")
			loc = n.Loc
		}
	case *ast.DeleteStmt:
		if n.Relation != nil {
			tableName = normalizeTableRef(n.Relation, "", "")
			loc = n.Loc
		}
	case *ast.MergeStmt:
		if n.Target != nil {
			tableName = normalizeTableRef(n.Target, "", "")
			loc = n.Loc
		}
	case *ast.SelectStmt:
		if n.IntoTable != nil {
			tableName = normalizeTableRef(n.IntoTable, "", "")
			loc = n.Loc
		}
	default:
		return
	}

	if tableName == "" {
		return
	}

	for _, disallow := range r.disallowList {
		if tableName == disallow {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.TableDisallowDML.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("DML is disallowed on table %s.", tableName),
				StartPosition: &storepb.Position{Line: r.LocToLine(loc)},
			})
			return
		}
	}
}
