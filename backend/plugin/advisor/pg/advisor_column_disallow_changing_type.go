package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleColumnDisallowChangeType, &ColumnDisallowChangingTypeAdvisor{})
}

// ColumnDisallowChangingTypeAdvisor is the advisor checking for disallow changing column type.
type ColumnDisallowChangingTypeAdvisor struct {
}

// Check checks for disallow changing column type.
func (*ColumnDisallowChangingTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnDisallowChangingTypeRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		tokens: tree.Tokens,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type columnDisallowChangingTypeRule struct {
	BaseRule

	tokens *antlr.CommonTokenStream
}

func (*columnDisallowChangingTypeRule) Name() string {
	return "column-disallow-changing-type"
}

func (r *columnDisallowChangingTypeRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Altertablestmt":
		r.handleAltertablestmt(ctx.(*parser.AltertablestmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*columnDisallowChangingTypeRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *columnDisallowChangingTypeRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check for ALTER COLUMN TYPE statements
	if ctx.Alter_table_cmds() == nil {
		return
	}

	allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
	for _, cmd := range allCmds {
		// ALTER opt_column? colid opt_set_data? TYPE_P typename ...
		if cmd.ALTER() != nil && cmd.TYPE_P() != nil {
			// This is an ALTER COLUMN TYPE statement
			text := r.tokens.GetTextFromRuleContext(ctx)

			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    code.ChangeColumnType.Int32(),
				Title:   r.title,
				Content: fmt.Sprintf("The statement \"%s\" changes column type", text),
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
			break // Only report once per ALTER TABLE statement
		}
	}
}
