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
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleTableNoFK, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &tableNoFKChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type tableNoFKChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
}

// EnterCreatestmt handles CREATE TABLE with FK constraints
func (c *tableNoFKChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var tableName, schemaName string
	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		tableName = extractTableName(allQualifiedNames[0])
		schemaName = extractSchemaName(allQualifiedNames[0])
		if schemaName == "" {
			schemaName = "public"
		}
	}

	// Check table-level constraints
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Check if this is a table constraint
			if elem.Tableconstraint() != nil {
				constraint := elem.Tableconstraint()
				if constraint.Constraintelem() != nil {
					constraintElem := constraint.Constraintelem()
					// Check for FOREIGN KEY
					if constraintElem.FOREIGN() != nil && constraintElem.KEY() != nil {
						c.addFKAdvice(schemaName, tableName, ctx)
						return
					}
				}
			}

			// Check column-level constraints
			if elem.ColumnDef() != nil {
				columnDef := elem.ColumnDef()
				if columnDef.Colquallist() != nil {
					allQuals := columnDef.Colquallist().AllColconstraint()
					for _, qual := range allQuals {
						if qual.Colconstraintelem() != nil {
							constraintElem := qual.Colconstraintelem()
							// Check for REFERENCES (column-level FK)
							if constraintElem.REFERENCES() != nil {
								c.addFKAdvice(schemaName, tableName, ctx)
								return
							}
						}
					}
				}
			}
		}
	}
}

// EnterAltertablestmt handles ALTER TABLE ADD CONSTRAINT with FK
func (c *tableNoFKChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var tableName, schemaName string
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		tableName = extractTableName(ctx.Relation_expr().Qualified_name())
		schemaName = extractSchemaName(ctx.Relation_expr().Qualified_name())
		if schemaName == "" {
			schemaName = "public"
		}
	}

	// Check all alter table commands for ADD CONSTRAINT with FOREIGN KEY
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				constraint := cmd.Tableconstraint()
				if constraint.Constraintelem() != nil {
					constraintElem := constraint.Constraintelem()
					// Check for FOREIGN KEY
					if constraintElem.FOREIGN() != nil && constraintElem.KEY() != nil {
						c.addFKAdvice(schemaName, tableName, ctx)
						return
					}
				}
			}
		}
	}
}

func (c *tableNoFKChecker) addFKAdvice(schemaName, tableName string, ctx antlr.ParserRuleContext) {
	stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:  c.level,
		Code:    advisor.TableHasFK.Int32(),
		Title:   c.title,
		Content: fmt.Sprintf("Foreign key is not allowed in the table %q.%q, related statement: \"%s\"", schemaName, tableName, stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
