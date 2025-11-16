package pg

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*ColumnDefaultDisallowVolatileAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleColumnDefaultDisallowVolatile, &ColumnDefaultDisallowVolatileAdvisor{})
}

// ColumnDefaultDisallowVolatileAdvisor is the advisor checking for column default volatile functions.
type ColumnDefaultDisallowVolatileAdvisor struct {
}

// Check checks for column default volatile functions.
func (*ColumnDefaultDisallowVolatileAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnDefaultDisallowVolatileRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		columnSet: make(map[string]columnData),
	}

	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type columnData struct {
	schema string
	table  string
	name   string
	line   int
}

type columnDefaultDisallowVolatileRule struct {
	BaseRule

	columnSet map[string]columnData
}

func (*columnDefaultDisallowVolatileRule) Name() string {
	return string(advisor.SchemaRuleColumnDefaultDisallowVolatile)
}

func (r *columnDefaultDisallowVolatileRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Altertablestmt":
		r.handleAltertablestmt(ctx.(*parser.AltertablestmtContext))
		// Do nothing for other node types
	default:
	}
	return nil
}

func (r *columnDefaultDisallowVolatileRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	// Generate advice when we exit the root node
	if nodeType == "Root" {
		r.generateAdvice()
	}
	return nil
}

func (r *columnDefaultDisallowVolatileRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := ctx.Relation_expr().Qualified_name().GetText()
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
				if colDef.Colid() != nil {
					columnName := pg.NormalizePostgreSQLColid(colDef.Colid())

					// Check if this column has a volatile DEFAULT
					if r.hasVolatileDefault(colDef) {
						r.addColumn("public", tableName, columnName, colDef.GetStart().GetLine())
					}
				}
			}
		}
	}
}

func (r *columnDefaultDisallowVolatileRule) hasVolatileDefault(colDef parser.IColumnDefContext) bool {
	if colDef == nil || colDef.Colquallist() == nil {
		return false
	}

	// Check all column constraints
	allConstraints := colDef.Colquallist().AllColconstraint()
	for _, constraint := range allConstraints {
		// Check if this is a DEFAULT constraint
		if constraint.Colconstraintelem() != nil {
			elem := constraint.Colconstraintelem()
			if elem.DEFAULT() != nil && elem.B_expr() != nil {
				// If the default expression contains a function call, it's potentially volatile
				// We check if the expression contains a function call by looking for FuncExpr patterns
				if r.containsFunctionCall(elem.B_expr()) {
					return true
				}
			}
		}
	}

	return false
}

func (r *columnDefaultDisallowVolatileRule) containsFunctionCall(expr antlr.Tree) bool {
	if expr == nil {
		return false
	}

	// Check if this expression is or contains a function call
	// In PostgreSQL, function calls are represented as func_expr
	return r.hasFuncExpr(expr)
}

func (r *columnDefaultDisallowVolatileRule) hasFuncExpr(node antlr.Tree) bool {
	if node == nil {
		return false
	}

	// Check if this node is a function expression
	switch node.(type) {
	case *parser.Func_exprContext,
		*parser.Func_expr_windowlessContext,
		*parser.Func_expr_common_subexprContext:
		return true
	}

	// Recursively check children
	if parserRule, ok := node.(antlr.ParserRuleContext); ok {
		for i := 0; i < parserRule.GetChildCount(); i++ {
			child := parserRule.GetChild(i)
			if r.hasFuncExpr(child) {
				return true
			}
		}
	}

	return false
}

func (r *columnDefaultDisallowVolatileRule) addColumn(schema string, table string, column string, line int) {
	if schema == "" {
		schema = "public"
	}

	r.columnSet[fmt.Sprintf("%s.%s.%s", schema, table, column)] = columnData{
		schema: schema,
		table:  table,
		name:   column,
		line:   line,
	}
}

func (r *columnDefaultDisallowVolatileRule) generateAdvice() {
	var columnList []columnData
	for _, column := range r.columnSet {
		columnList = append(columnList, column)
	}
	slices.SortFunc(columnList, func(i, j columnData) int {
		if i.line < j.line {
			return -1
		}
		if i.line > j.line {
			return 1
		}
		return 0
	})

	for _, column := range columnList {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NoDefault.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("Column %q.%q in schema %q has volatile DEFAULT", column.table, column.name, column.schema),
			StartPosition: &storepb.Position{
				Line:   int32(column.line),
				Column: 0,
			},
		})
	}
}
