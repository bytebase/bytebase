package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*NamingTableNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD, &NamingTableNoKeywordAdvisor{})
}

// NamingTableNoKeywordAdvisor is the advisor checking for table naming convention without keyword.
type NamingTableNoKeywordAdvisor struct {
}

// Check checks for table naming convention without keyword.
func (*NamingTableNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewNamingTableNoKeywordRule(level, checkCtx.Rule.Type.String())

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

// NamingTableNoKeywordRule is the rule for table naming convention without keyword.
type NamingTableNoKeywordRule struct {
	BaseRule
}

// NewNamingTableNoKeywordRule creates a new NamingTableNoKeywordRule.
func NewNamingTableNoKeywordRule(level storepb.Advice_Status, title string) *NamingTableNoKeywordRule {
	return &NamingTableNoKeywordRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*NamingTableNoKeywordRule) Name() string {
	return "NamingTableNoKeywordRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingTableNoKeywordRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeCreateTable {
		r.enterCreateTable(ctx.(*parser.Create_tableContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NamingTableNoKeywordRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *NamingTableNoKeywordRule) enterCreateTable(ctx *parser.Create_tableContext) {
	tableName := ctx.Table_name().GetTable()
	_, normalizedTableName := tsqlparser.NormalizeTSQLIdentifier(tableName)
	if tsqlparser.IsTSQLReservedKeyword(normalizedTableName, false) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NameIsKeywordIdentifier.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Table name [%s] is a reserved keyword and should be avoided.", normalizedTableName),
			StartPosition: common.ConvertANTLRLineToPosition(tableName.GetStart().GetLine()),
		})
	}
}
