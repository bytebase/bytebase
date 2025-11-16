package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleColumnTypeDisallowList, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for column type restriction.
type ColumnTypeDisallowListAdvisor struct {
}

// Check checks for column type restriction.
func (*ColumnTypeDisallowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Convert disallowed types to lowercase for case-insensitive comparison
	typeRestriction := make(map[string]bool)
	for _, tp := range payload.List {
		typeRestriction[strings.ToLower(tp)] = true
	}

	rule := &columnTypeDisallowListRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		typeRestriction: typeRestriction,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type columnTypeDisallowListRule struct {
	BaseRule

	typeRestriction map[string]bool
}

func (*columnTypeDisallowListRule) Name() string {
	return "column_type_disallow_list"
}

func (r *columnTypeDisallowListRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		if c, ok := ctx.(*parser.CreatestmtContext); ok {
			r.handleCreatestmt(c)
		}
	case "Altertablestmt":
		if c, ok := ctx.(*parser.AltertablestmtContext); ok {
			r.handleAltertablestmt(c)
		}
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*columnTypeDisallowListRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *columnTypeDisallowListRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	qualifiedNames := ctx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}

	tableName := extractTableName(qualifiedNames[0])
	if tableName == "" {
		return
	}

	// Check all columns for disallowed types
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.ColumnDef() != nil {
				colDef := elem.ColumnDef()
				if colDef.Colid() != nil && colDef.Typename() != nil {
					columnName := pg.NormalizePostgreSQLColid(colDef.Colid())
					r.checkType(tableName, columnName, colDef.Typename(), colDef.GetStart().GetLine())
				}
			}
		}
	}
}

func (r *columnTypeDisallowListRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := extractTableName(ctx.Relation_expr().Qualified_name())
	if tableName == "" {
		return
	}

	// Check ALTER TABLE ADD COLUMN
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD COLUMN
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
				colDef := cmd.ColumnDef()
				if colDef.Colid() != nil && colDef.Typename() != nil {
					columnName := pg.NormalizePostgreSQLColid(colDef.Colid())
					r.checkType(tableName, columnName, colDef.Typename(), colDef.GetStart().GetLine())
				}
			}

			// ALTER COLUMN TYPE
			if cmd.ALTER() != nil && cmd.TYPE_P() != nil && cmd.Typename() != nil {
				allColids := cmd.AllColid()
				if len(allColids) > 0 {
					columnName := pg.NormalizePostgreSQLColid(allColids[0])
					r.checkType(tableName, columnName, cmd.Typename(), cmd.GetStart().GetLine())
				}
			}
		}
	}
}

func (r *columnTypeDisallowListRule) checkType(tableName, columnName string, typename parser.ITypenameContext, line int) {
	if typename == nil {
		return
	}

	// Get the type text
	typeText := typename.GetText()

	// Check if this type is equivalent to any type in the disallow list
	var matchedDisallowedType string
	for disallowedType := range r.typeRestriction {
		if areTypesEquivalent(typeText, disallowedType) {
			matchedDisallowedType = disallowedType
			break
		}
	}

	if matchedDisallowedType != "" {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.DisabledColumnType.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("Disallow column type %s but column %q.%q is", strings.ToUpper(matchedDisallowedType), tableName, columnName),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}
