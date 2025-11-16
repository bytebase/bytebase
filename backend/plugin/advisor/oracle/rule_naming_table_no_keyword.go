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
	_ advisor.Advisor = (*NamingTableNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleTableNameNoKeyword, &NamingTableNoKeywordAdvisor{})
}

// NamingTableNoKeywordAdvisor is the advisor checking for table naming convention without keyword.
type NamingTableNoKeywordAdvisor struct {
}

// Check checks for table naming convention without keyword.
func (*NamingTableNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewNamingTableNoKeywordRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// NamingTableNoKeywordRule is the rule implementation for table naming convention without keyword.
type NamingTableNoKeywordRule struct {
	BaseRule

	currentDatabase string
}

// NewNamingTableNoKeywordRule creates a new NamingTableNoKeywordRule.
func NewNamingTableNoKeywordRule(level storepb.Advice_Status, title string, currentDatabase string) *NamingTableNoKeywordRule {
	return &NamingTableNoKeywordRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*NamingTableNoKeywordRule) Name() string {
	return "naming.table-no-keyword"
}

// OnEnter is called when the parser enters a rule context.
func (r *NamingTableNoKeywordRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.handleCreateTable(ctx.(*parser.Create_tableContext))
	case "Alter_table_properties":
		r.handleAlterTableProperties(ctx.(*parser.Alter_table_propertiesContext))
	default:
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*NamingTableNoKeywordRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *NamingTableNoKeywordRule) handleCreateTable(ctx *parser.Create_tableContext) {
	tableName := normalizeIdentifier(ctx.Table_name(), r.currentDatabase)
	if plsqlparser.IsOracleKeyword(tableName) {
		r.AddAdvice(
			r.level,
			code.NameIsKeywordIdentifier.Int32(),
			fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", tableName),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
}

func (r *NamingTableNoKeywordRule) handleAlterTableProperties(ctx *parser.Alter_table_propertiesContext) {
	if ctx.Tableview_name() == nil {
		return
	}
	tableName := lastIdentifier(normalizeIdentifier(ctx.Tableview_name(), r.currentDatabase))
	if plsqlparser.IsOracleKeyword(tableName) {
		r.AddAdvice(
			r.level,
			code.NameIsKeywordIdentifier.Int32(),
			fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", tableName),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
}
