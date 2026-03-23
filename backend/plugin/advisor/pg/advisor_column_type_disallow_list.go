package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for column type restriction.
type ColumnTypeDisallowListAdvisor struct {
}

// Check checks for column type restriction.
func (*ColumnTypeDisallowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	typeRestriction := make(map[string]bool)
	for _, tp := range stringArrayPayload.List {
		typeRestriction[strings.ToLower(tp)] = true
	}

	rule := &columnTypeDisallowListRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		typeRestriction: typeRestriction,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnTypeDisallowListRule struct {
	OmniBaseRule

	typeRestriction map[string]bool
}

func (*columnTypeDisallowListRule) Name() string {
	return "column_type_disallow_list"
}

func (r *columnTypeDisallowListRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	default:
	}
}

func (r *columnTypeDisallowListRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}

	cols, _ := omniTableElements(n)
	for _, col := range cols {
		if col.TypeName != nil {
			r.checkType(tableName, col.Colname, col.TypeName, col.Colname)
		}
	}
}

func (r *columnTypeDisallowListRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}

	for _, cmd := range omniAlterTableCmds(n) {
		switch ast.AlterTableType(cmd.Subtype) {
		case ast.AT_AddColumn:
			colDef, ok := cmd.Def.(*ast.ColumnDef)
			if !ok || colDef == nil || colDef.TypeName == nil {
				continue
			}
			r.checkType(tableName, colDef.Colname, colDef.TypeName, colDef.Colname)
		case ast.AT_AlterColumnType:
			typeName, ok := cmd.Def.(*ast.ColumnDef)
			if !ok || typeName == nil || typeName.TypeName == nil {
				continue
			}
			r.checkType(tableName, cmd.Name, typeName.TypeName, cmd.Name)
		default:
		}
	}
}

func (r *columnTypeDisallowListRule) checkType(tableName, columnName string, tn *ast.TypeName, searchName string) {
	// Build a type text from the TypeName for equivalence checking
	typeText := omniTypeName(tn)

	var matchedDisallowedType string
	for disallowedType := range r.typeRestriction {
		if areTypesEquivalent(typeText, disallowedType) {
			matchedDisallowedType = disallowedType
			break
		}
	}

	if matchedDisallowedType != "" {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.DisabledColumnType.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Disallow column type %s but column %q.%q is", strings.ToUpper(matchedDisallowedType), tableName, columnName),
			StartPosition: &storepb.Position{
				Line:   r.FindLineByName(searchName),
				Column: 0,
			},
		})
	}
}
