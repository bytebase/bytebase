package pgantlr

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*StatementDisallowAddColumnWithDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementDisallowAddColumnWithDefault, &StatementDisallowAddColumnWithDefaultAdvisor{})
}

// StatementDisallowAddColumnWithDefaultAdvisor is the advisor checking for to disallow add column with default.
type StatementDisallowAddColumnWithDefaultAdvisor struct {
}

// Check checks for to disallow add column with default.
func (*StatementDisallowAddColumnWithDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementDisallowAddColumnWithDefaultChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementDisallowAddColumnWithDefaultChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// EnterAltertablestmt handles ALTER TABLE ADD COLUMN
func (c *statementDisallowAddColumnWithDefaultChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check all alter table commands
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// Check for ADD COLUMN
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
				columnDef := cmd.ColumnDef()
				// Check if the column has a DEFAULT constraint
				if columnDef.Colquallist() != nil {
					allConstraints := columnDef.Colquallist().AllColconstraint()
					for _, constraint := range allConstraints {
						if constraint.Colconstraintelem() != nil {
							constraintElem := constraint.Colconstraintelem()
							// Check for DEFAULT constraint
							if constraintElem.DEFAULT() != nil {
								c.adviceList = append(c.adviceList, &storepb.Advice{
									Status:  c.level,
									Code:    advisor.StatementAddColumnWithDefault.Int32(),
									Title:   c.title,
									Content: "Adding column with DEFAULT will locked the whole table and rewriting each rows",
									StartPosition: &storepb.Position{
										Line:   int32(ctx.GetStart().GetLine()),
										Column: 0,
									},
								})
								return
							}
						}
					}
				}
			}
		}
	}
}
