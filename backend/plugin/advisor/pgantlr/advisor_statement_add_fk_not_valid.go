package pgantlr

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*StatementAddFKNotValidAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementAddFKNotValid, &StatementAddFKNotValidAdvisor{})
}

// StatementAddFKNotValidAdvisor is the advisor checking for adding foreign key constraints without NOT VALID.
type StatementAddFKNotValidAdvisor struct {
}

// Check checks for adding foreign key constraints without NOT VALID.
func (*StatementAddFKNotValidAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementAddFKNotValidChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementAddFKNotValidChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// EnterAltertablestmt handles ALTER TABLE ADD CONSTRAINT FOREIGN KEY
func (c *statementAddFKNotValidChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Alter_table_cmds() == nil {
		return
	}

	allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
	for _, cmd := range allCmds {
		// Check for ADD + something
		if cmd.ADD_P() == nil {
			continue
		}

		// Check for Tableconstraint
		if cmd.Tableconstraint() == nil {
			continue
		}

		constraint := cmd.Tableconstraint()
		if constraint.Constraintelem() == nil {
			continue
		}

		elem := constraint.Constraintelem()

		// Check if this is a FOREIGN KEY constraint
		if elem.FOREIGN() == nil || elem.KEY() == nil {
			continue
		}

		// Check if NOT VALID is present
		hasNotValid := false
		if elem.Constraintattributespec() != nil {
			allAttrs := elem.Constraintattributespec().AllConstraintattributeElem()
			for _, attr := range allAttrs {
				if attr.NOT() != nil && attr.VALID() != nil {
					hasNotValid = true
					break
				}
			}
		}

		// If NOT VALID is not present, this is a problem
		if !hasNotValid {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Code:    advisor.StatementAddFKWithValidation.Int32(),
				Title:   c.title,
				Content: "Adding foreign keys with validation will block reads and writes. You can add check foreign keys not valid and then validate separately",
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}
