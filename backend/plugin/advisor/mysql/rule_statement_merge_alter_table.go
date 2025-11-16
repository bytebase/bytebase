package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementMergeAlterTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementMergeAlterTable, &StatementMergeAlterTableAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementMergeAlterTable, &StatementMergeAlterTableAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleStatementMergeAlterTable, &StatementMergeAlterTableAdvisor{})
}

// StatementMergeAlterTableAdvisor is the advisor checking for merging ALTER TABLE statements.
type StatementMergeAlterTableAdvisor struct {
}

// Check checks for merging ALTER TABLE statements.
func (*StatementMergeAlterTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementMergeAlterTableRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	// Generate advice based on collected table information
	rule.generateAdvice()

	return checker.GetAdviceList(), nil
}

// tableStatement represents information about a table's statements.
type tableStatement struct {
	name     string
	count    int
	lastLine int
}

// StatementMergeAlterTableRule checks for mergeable ALTER TABLE statements.
type StatementMergeAlterTableRule struct {
	BaseRule
	text     string
	tableMap map[string]tableStatement
}

// NewStatementMergeAlterTableRule creates a new StatementMergeAlterTableRule.
func NewStatementMergeAlterTableRule(level storepb.Advice_Status, title string) *StatementMergeAlterTableRule {
	return &StatementMergeAlterTableRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		tableMap: make(map[string]tableStatement),
	}
}

// Name returns the rule name.
func (*StatementMergeAlterTableRule) Name() string {
	return "StatementMergeAlterTableRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementMergeAlterTableRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementMergeAlterTableRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *StatementMergeAlterTableRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	r.tableMap[tableName] = tableStatement{
		name:     tableName,
		count:    1,
		lastLine: r.baseLine + ctx.GetStart().GetLine(),
	}
}

func (r *StatementMergeAlterTableRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRef() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	table, ok := r.tableMap[tableName]
	if !ok {
		table = tableStatement{
			name:  tableName,
			count: 0,
		}
	}
	table.count++
	table.lastLine = r.baseLine + ctx.GetStart().GetLine()
	r.tableMap[tableName] = table
}

func (r *StatementMergeAlterTableRule) generateAdvice() {
	var tableList []tableStatement
	for _, table := range r.tableMap {
		tableList = append(tableList, table)
	}
	slices.SortFunc(tableList, func(i, j tableStatement) int {
		if i.lastLine < j.lastLine {
			return -1
		}
		if i.lastLine > j.lastLine {
			return 1
		}
		return 0
	})

	for _, table := range tableList {
		if table.count > 1 {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.StatementRedundantAlterTable.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("There are %d statements to modify table `%s`", table.count, table.name),
				StartPosition: common.ConvertANTLRLineToPosition(table.lastLine),
			})
		}
	}
}
