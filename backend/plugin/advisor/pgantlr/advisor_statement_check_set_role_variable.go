package pgantlr

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*StatementCheckSetRoleVariable)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementCheckSetRoleVariable, &StatementCheckSetRoleVariable{})
}

type StatementCheckSetRoleVariable struct {
}

func (*StatementCheckSetRoleVariable) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementCheckSetRoleVariableChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	if !checker.hasSetRole {
		return []*storepb.Advice{{
			Status:        level,
			Code:          advisor.StatementCheckSetRoleVariable.Int32(),
			Title:         checker.title,
			Content:       "No SET ROLE statement found.",
			StartPosition: nil,
		}}, nil
	}

	return nil, nil
}

type statementCheckSetRoleVariableChecker struct {
	*parser.BasePostgreSQLParserListener

	level           storepb.Advice_Status
	title           string
	hasSetRole      bool
	foundNonSetStmt bool
}

// EnterVariablesetstmt handles SET statements
func (c *statementCheckSetRoleVariableChecker) EnterVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// If we already found a non-SET statement, skip this
	if c.foundNonSetStmt {
		return
	}

	// Check if this is a SET ROLE statement
	setRest := ctx.Set_rest()
	if setRest != nil {
		setRestMore := setRest.Set_rest_more()
		if setRestMore != nil && setRestMore.ROLE() != nil {
			c.hasSetRole = true
		}
	}
}

// EnterEveryRule is called for every rule entry to detect non-SET statements
func (c *statementCheckSetRoleVariableChecker) EnterEveryRule(ctx antlr.ParserRuleContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// If we already found a non-SET statement, no need to continue checking
	if c.foundNonSetStmt {
		return
	}

	// Check if this is a non-SET statement at the top level
	// We only care about statements that are not VariablesetstmtContext
	if _, isSetStmt := ctx.(*parser.VariablesetstmtContext); !isSetStmt {
		// Check if this is a statement node (not structural nodes like Stmt, Root, etc.)
		switch ctx.(type) {
		case *parser.RootContext, *parser.StmtblockContext, *parser.StmtmultiContext, *parser.StmtContext:
			// These are structural nodes, not actual statements
			return
		default:
			// This is a non-SET statement
			c.foundNonSetStmt = true
		}
	}
}
