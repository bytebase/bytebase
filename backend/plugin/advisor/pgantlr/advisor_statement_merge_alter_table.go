package pgantlr

import (
	"context"
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*StatementMergeAlterTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementMergeAlterTable, &StatementMergeAlterTableAdvisor{})
}

// StatementMergeAlterTableAdvisor is the advisor checking for no redundant ALTER TABLE statements.
type StatementMergeAlterTableAdvisor struct {
}

// Check checks for no redundant ALTER TABLE statements.
func (*StatementMergeAlterTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementMergeAlterTableChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		tableMap:                     make(tableMap),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.generateAdvice(), nil
}

type statementMergeAlterTableChecker struct {
	*parser.BasePostgreSQLParserListener

	level    storepb.Advice_Status
	title    string
	tableMap tableMap
}

type tableMap map[string]tableStatement

type tableStatement struct {
	schema string
	name   string
	count  int
	line   int
}

func (m tableMap) set(schema string, table string, line int) {
	t := tableStatement{
		schema: schema,
		name:   table,
		count:  1,
		line:   line,
	}
	m[t.key()] = t
}

func (m tableMap) add(schema string, table string, line int) {
	if t, exists := m[fmt.Sprintf("%s.%s", schema, table)]; exists {
		t.count++
		t.line = line
		m[t.key()] = t
	}
}

func (t tableStatement) key() string {
	return fmt.Sprintf("%s.%s", t.schema, t.name)
}

func (checker *statementMergeAlterTableChecker) generateAdvice() []*storepb.Advice {
	var adviceList []*storepb.Advice
	var tableList []tableStatement
	for _, table := range checker.tableMap {
		tableList = append(tableList, table)
	}
	slices.SortFunc(tableList, func(i, j tableStatement) int {
		if i.line < j.line {
			return -1
		}
		if i.line > j.line {
			return 1
		}
		return 0
	})
	for _, table := range tableList {
		if table.count > 1 {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  checker.level,
				Code:    advisor.StatementRedundantAlterTable.Int32(),
				Title:   checker.title,
				Content: fmt.Sprintf("There are %d statements to modify table `%s`", table.count, table.name),
				StartPosition: &storepb.Position{
					Line:   int32(table.line),
					Column: 0,
				},
			})
		}
	}

	return adviceList
}

// EnterCreatestmt is called when production createstmt is entered.
func (checker *statementMergeAlterTableChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) == 0 {
		return
	}

	qualifiedName := allQualifiedNames[0]
	tableName := extractTableName(qualifiedName)
	schema := extractSchemaName(qualifiedName)
	if schema == "" {
		schema = "public"
	}

	if tableName == "" {
		return
	}

	checker.tableMap.set(schema, tableName, ctx.GetStop().GetLine())
}

// EnterAltertablestmt is called when production altertablestmt is entered.
func (checker *statementMergeAlterTableChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	relationExpr := ctx.Relation_expr()
	if relationExpr == nil {
		return
	}

	qualifiedName := relationExpr.Qualified_name()
	if qualifiedName == nil {
		return
	}

	tableName := extractTableName(qualifiedName)
	schema := extractSchemaName(qualifiedName)
	if schema == "" {
		schema = "public"
	}

	if tableName == "" {
		return
	}

	checker.tableMap.add(schema, tableName, ctx.GetStop().GetLine())
}
