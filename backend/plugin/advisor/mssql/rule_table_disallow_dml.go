package mssql

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

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
	_ advisor.Advisor = (*TableDisallowDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DML, &TableDisallowDMLAdvisor{})
}

// TableDisallowDMLAdvisor is the advisor checking for disallow DML on specific tables.
type TableDisallowDMLAdvisor struct {
}

func (*TableDisallowDMLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for table disallow DML rule")
	}

	// Create the rule
	rule := NewTableDisallowDMLRule(level, checkCtx.Rule.Type.String(), stringArrayPayload.List)

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

// TableDisallowDMLRule is the rule checking for disallow DML on specific tables.
type TableDisallowDMLRule struct {
	BaseRule
	// disallowList is the list of table names that disallow DML.
	disallowList []string
}

// NewTableDisallowDMLRule creates a new TableDisallowDMLRule.
func NewTableDisallowDMLRule(level storepb.Advice_Status, title string, disallowList []string) *TableDisallowDMLRule {
	return &TableDisallowDMLRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		disallowList: disallowList,
	}
}

// Name returns the rule name.
func (*TableDisallowDMLRule) Name() string {
	return "TableDisallowDMLRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableDisallowDMLRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Merge_statement":
		r.enterMergeStatement(ctx.(*parser.Merge_statementContext))
	case NodeTypeInsertStatement:
		r.enterInsertStatement(ctx.(*parser.Insert_statementContext))
	case NodeTypeDeleteStatement:
		r.enterDeleteStatement(ctx.(*parser.Delete_statementContext))
	case NodeTypeUpdateStatement:
		r.enterUpdateStatement(ctx.(*parser.Update_statementContext))
	case "Select_statement_standalone":
		r.enterSelectStatementStandalone(ctx.(*parser.Select_statement_standaloneContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableDisallowDMLRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *TableDisallowDMLRule) enterMergeStatement(ctx *parser.Merge_statementContext) {
	if ctx.Ddl_object() == nil {
		return
	}
	tableName := ctx.Ddl_object().GetText()
	r.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDMLRule) enterInsertStatement(ctx *parser.Insert_statementContext) {
	if ctx.Ddl_object() == nil {
		return
	}
	tableName := ctx.Ddl_object().GetText()
	r.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDMLRule) enterDeleteStatement(ctx *parser.Delete_statementContext) {
	if ctx.Delete_statement_from() == nil {
		return
	}
	tableName := ctx.Delete_statement_from().GetText()
	r.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDMLRule) enterUpdateStatement(ctx *parser.Update_statementContext) {
	if ctx.Ddl_object() == nil {
		return
	}
	tableName := ctx.Ddl_object().GetText()
	r.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDMLRule) enterSelectStatementStandalone(ctx *parser.Select_statement_standaloneContext) {
	querySpec := ctx.Select_statement().Query_expression().Query_specification()
	if querySpec == nil {
		return
	}
	if querySpec.INTO() == nil || querySpec.Table_name() == nil {
		return
	}
	tableName := tsqlparser.NormalizeTSQLTableName(querySpec.Table_name(), "" /* fallbackDatabase */, "" /* fallbackSchema */, false /* caseSensitive */)
	r.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDMLRule) checkTableName(normalizedTableName string, line int) {
	for _, disallow := range r.disallowList {
		if normalizedTableName == disallow {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.TableDisallowDML.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("DML is disallowed on table %s.", normalizedTableName),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
			return
		}
	}
}
