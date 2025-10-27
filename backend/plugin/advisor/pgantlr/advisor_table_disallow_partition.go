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
	_ advisor.Advisor = (*TableDisallowPartitionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleTableDisallowPartition, &TableDisallowPartitionAdvisor{})
}

// TableDisallowPartitionAdvisor is the advisor checking for disallow table partition.
type TableDisallowPartitionAdvisor struct {
}

// Check checks for disallow table partition.
func (*TableDisallowPartitionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &tableDisallowPartitionChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type tableDisallowPartitionChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
}

// EnterCreatestmt handles CREATE TABLE statements
func (c *tableDisallowPartitionChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this CREATE TABLE has a PARTITION BY clause
	if ctx.Optpartitionspec() != nil && ctx.Optpartitionspec().Partitionspec() != nil {
		stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.CreateTablePartition.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", stmtText),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// EnterPartition_cmd handles ALTER TABLE ... ATTACH PARTITION
func (c *tableDisallowPartitionChecker) EnterPartition_cmd(ctx *parser.Partition_cmdContext) {
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
				stmtText := extractStatementText(c.statementsText, alterTableCtx.GetStart().GetLine(), alterTableCtx.GetStop().GetLine())
				c.adviceList = append(c.adviceList, &storepb.Advice{
					Status:  c.level,
					Code:    advisor.CreateTablePartition.Int32(),
					Title:   c.title,
					Content: fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", stmtText),
					StartPosition: &storepb.Position{
						Line:   int32(alterTableCtx.GetStart().GetLine()),
						Column: 0,
					},
				})
				break
			}
			parent = parent.GetParent()
		}
	}
}
