package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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

	rule := &statementDisallowAddNotNullRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type statementDisallowAddNotNullRule struct {
	BaseRule
}

func (*statementDisallowAddNotNullRule) Name() string {
	return "statement_disallow_add_not_null"
}

func (r *statementDisallowAddNotNullRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Altertablestmt":
		r.handleAltertablestmt(ctx)
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*statementDisallowAddNotNullRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleAltertablestmt handles ALTER TABLE ALTER COLUMN SET NOT NULL
func (r *statementDisallowAddNotNullRule) handleAltertablestmt(ctx antlr.ParserRuleContext) {
	altertablestmtCtx, ok := ctx.(*parser.AltertablestmtContext)
	if !ok {
		return
	}

	if !isTopLevel(altertablestmtCtx.GetParent()) {
		return
	}

	// Check all alter table commands
	if altertablestmtCtx.Alter_table_cmds() != nil {
		allCmds := altertablestmtCtx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// Check for ALTER COLUMN ... SET NOT NULL
			if cmd.ALTER() != nil && cmd.SET() != nil && cmd.NOT() != nil && cmd.NULL_P() != nil {
				// Get the column name
				allColIDs := cmd.AllColid()
				if len(allColIDs) > 0 {
					columnName := pgparser.NormalizePostgreSQLColid(allColIDs[0])
					r.AddAdvice(&storepb.Advice{
						Status:  r.level,
						Code:    code.StatementAddNotNull.Int32(),
						Title:   r.title,
						Content: fmt.Sprintf("Setting NOT NULL will block reads and writes. You can use CHECK (%q IS NOT NULL) instead", columnName),
						StartPosition: &storepb.Position{
							Line:   int32(altertablestmtCtx.GetStart().GetLine()),
							Column: 0,
						},
					})
				}
			}
		}
	}
}
