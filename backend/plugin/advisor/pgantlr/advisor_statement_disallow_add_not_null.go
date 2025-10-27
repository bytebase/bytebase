package pgantlr

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*StatementDisallowAddNotNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementDisallowAddNotNull, &StatementDisallowAddNotNullAdvisor{})
}

// StatementDisallowAddNotNullAdvisor is the advisor checking for to disallow add not null.
type StatementDisallowAddNotNullAdvisor struct {
}

// Check checks for to disallow add not null.
func (*StatementDisallowAddNotNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementDisallowAddNotNullChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementDisallowAddNotNullChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// EnterAltertablestmt handles ALTER TABLE ALTER COLUMN SET NOT NULL
func (c *statementDisallowAddNotNullChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check all alter table commands
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// Check for ALTER COLUMN ... SET NOT NULL
			if cmd.ALTER() != nil && cmd.SET() != nil && cmd.NOT() != nil && cmd.NULL_P() != nil {
				// Get the column name
				allColIDs := cmd.AllColid()
				if len(allColIDs) > 0 {
					columnName := pgparser.NormalizePostgreSQLColid(allColIDs[0])
					c.adviceList = append(c.adviceList, &storepb.Advice{
						Status:  c.level,
						Code:    advisor.StatementAddNotNull.Int32(),
						Title:   c.title,
						Content: fmt.Sprintf("Setting NOT NULL will block reads and writes. You can use CHECK (%q IS NOT NULL) instead", columnName),
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
