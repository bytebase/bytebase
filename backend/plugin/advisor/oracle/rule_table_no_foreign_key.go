// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*TableNoForeignKeyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleTableNoFK, &TableNoForeignKeyAdvisor{})
}

// TableNoForeignKeyAdvisor is the advisor checking for table disallow foreign key.
type TableNoForeignKeyAdvisor struct {
}

// Check checks for table disallow foreign key.
func (*TableNoForeignKeyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewTableNoForeignKeyRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// TableNoForeignKeyRule is the rule implementation for table disallow foreign key.
type TableNoForeignKeyRule struct {
	BaseRule

	currentDatabase string
	tableName       string
	tableWithFK     map[string]bool
	tableLine       map[string]int
}

// NewTableNoForeignKeyRule creates a new TableNoForeignKeyRule.
func NewTableNoForeignKeyRule(level storepb.Advice_Status, title string, currentDatabase string) *TableNoForeignKeyRule {
	return &TableNoForeignKeyRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		tableWithFK:     make(map[string]bool),
		tableLine:       make(map[string]int),
	}
}

// Name returns the rule name.
func (*TableNoForeignKeyRule) Name() string {
	return "table.no-foreign-key"
}

// OnEnter is called when the parser enters a rule context.
func (r *TableNoForeignKeyRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.handleCreateTable(ctx.(*parser.Create_tableContext))
	case "References_clause":
		r.handleReferencesClause(ctx.(*parser.References_clauseContext))
	case "Alter_table":
		r.handleAlterTable(ctx.(*parser.Alter_tableContext))
	default:
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (r *TableNoForeignKeyRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.tableName = ""
	case "Alter_table":
		r.tableName = ""
	default:
	}
	return nil
}

// GetAdviceList returns the advice list.
func (r *TableNoForeignKeyRule) GetAdviceList() ([]*storepb.Advice, error) {
	for tableName, hasFK := range r.tableWithFK {
		if hasFK {
			r.AddAdvice(
				r.level,
				code.TableHasFK.Int32(),
				fmt.Sprintf("Foreign key is not allowed in the table %s.", normalizeIdentifierName(tableName)),
				common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
			)
		}
	}
	return r.BaseRule.GetAdviceList()
}

func (r *TableNoForeignKeyRule) handleCreateTable(ctx *parser.Create_tableContext) {
	schemaName := r.currentDatabase
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), r.currentDatabase)
	}

	r.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), r.currentDatabase))
}

func (r *TableNoForeignKeyRule) handleReferencesClause(ctx *parser.References_clauseContext) {
	r.tableWithFK[r.tableName] = true
	r.tableLine[r.tableName] = r.baseLine + ctx.GetStop().GetLine()
}

func (r *TableNoForeignKeyRule) handleAlterTable(ctx *parser.Alter_tableContext) {
	r.tableName = normalizeIdentifier(ctx.Tableview_name(), r.currentDatabase)
}
