package pgantlr

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*ColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleColumnRequireDefault, &ColumnRequireDefaultAdvisor{})
}

// ColumnRequireDefaultAdvisor is the advisor checking for column default requirement.
type ColumnRequireDefaultAdvisor struct {
}

// Check checks for column default requirement.
func (*ColumnRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &columnRequireDefaultChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type columnRequireDefaultChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

func (c *columnRequireDefaultChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
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

	// Check all columns for DEFAULT clause
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.ColumnDef() != nil {
				colDef := elem.ColumnDef()
				if colDef.Colid() != nil {
					columnName := normalizeColid(colDef.Colid())
					// Check if column has DEFAULT
					if !c.hasDefault(colDef) {
						c.adviceList = append(c.adviceList, &storepb.Advice{
							Status:  c.level,
							Code:    advisor.NoDefault.Int32(),
							Title:   c.title,
							Content: fmt.Sprintf("Column %q.%q in schema %q doesn't have DEFAULT", tableName, columnName, "public"),
							StartPosition: &storepb.Position{
								Line:   int32(elem.GetStart().GetLine()),
								Column: 0,
							},
						})
					}
				}
			}
		}
	}
}

func (c *columnRequireDefaultChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
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
				if colDef.Colid() != nil {
					columnName := normalizeColid(colDef.Colid())
					// Check if column has DEFAULT
					if !c.hasDefault(colDef) {
						c.adviceList = append(c.adviceList, &storepb.Advice{
							Status:  c.level,
							Code:    advisor.NoDefault.Int32(),
							Title:   c.title,
							Content: fmt.Sprintf("Column %q.%q in schema %q doesn't have DEFAULT", tableName, columnName, "public"),
							StartPosition: &storepb.Position{
								Line:   int32(ctx.GetStart().GetLine()),
								Column: 0,
							},
						})
					}
				}
			}
		}
	}
}

// hasDefault checks if a column definition has a DEFAULT clause
// or uses a type that implicitly includes a default (like serial, bigserial, smallserial)
func (*columnRequireDefaultChecker) hasDefault(colDef parser.IColumnDefContext) bool {
	// Check if the type is serial/bigserial/smallserial (which have implicit defaults)
	if colDef.Typename() != nil && colDef.Typename().Simpletypename() != nil {
		simpleType := colDef.Typename().Simpletypename()
		typeText := normalizeTypeName(simpleType.GetText())
		// serial, bigserial, smallserial all have implicit DEFAULT nextval()
		if typeText == "serial" || typeText == "bigserial" || typeText == "smallserial" {
			return true
		}
	}

	// Check for explicit DEFAULT constraint
	if colDef.Colquallist() == nil {
		return false
	}

	allConstraints := colDef.Colquallist().AllColconstraint()
	for _, constraint := range allConstraints {
		if constraint.Colconstraintelem() == nil {
			continue
		}

		elem := constraint.Colconstraintelem()

		// Check for DEFAULT constraint
		if elem.DEFAULT() != nil {
			return true
		}
	}

	return false
}
