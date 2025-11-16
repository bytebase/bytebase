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
	_ advisor.Advisor = (*TableDisallowPartitionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleTableDisallowPartition, &TableDisallowPartitionAdvisor{})
}

// TableDisallowPartitionAdvisor is the advisor checking for partitioned tables.
type TableDisallowPartitionAdvisor struct {
}

// Check checks for partitioned tables.
func (*TableDisallowPartitionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableDisallowPartitionRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		statementsText: checkCtx.Statements,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type tableDisallowPartitionRule struct {
	BaseRule
	statementsText string
}

// Name returns the rule name.
func (*tableDisallowPartitionRule) Name() string {
	return "table.disallow-partition"
}

// OnEnter is called when the parser enters a rule context.
func (r *tableDisallowPartitionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx.(*parser.CreatestmtContext))
	case "Partition_cmd":
		r.handlePartitionCmd(ctx.(*parser.Partition_cmdContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*tableDisallowPartitionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *tableDisallowPartitionRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is a partitioned table
	if ctx.Optpartitionspec() != nil {
		stmtText := extractStatementText(r.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())

		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.CreateTablePartition.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("Table partition is forbidden, but %q creates", stmtText),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

func (r *tableDisallowPartitionRule) handlePartitionCmd(ctx *parser.Partition_cmdContext) {
	if !isTopLevel(ctx.GetParent().GetParent().GetParent()) {
		// Partition_cmd is nested: Altertablestmt -> Alter_table_cmds -> Alter_table_cmd -> Partition_cmd
		return
	}

	// Check for ATTACH PARTITION
	if ctx.ATTACH() != nil && ctx.PARTITION() != nil {
		// Navigate up to get the Altertablestmt context for statement text
		parent := ctx.GetParent()
		for parent != nil {
			if alterTableCtx, ok := parent.(*parser.AltertablestmtContext); ok {
				stmtText := extractStatementText(r.statementsText, alterTableCtx.GetStart().GetLine(), alterTableCtx.GetStop().GetLine())
				r.AddAdvice(&storepb.Advice{
					Status:  r.level,
					Code:    code.CreateTablePartition.Int32(),
					Title:   r.title,
					Content: fmt.Sprintf("Table partition is forbidden, but %q creates", stmtText),
					StartPosition: &storepb.Position{
						Line:   int32(alterTableCtx.GetStart().GetLine()),
						Column: 0,
					},
				})
				return
			}
			parent = parent.GetParent()
		}
	}
}
