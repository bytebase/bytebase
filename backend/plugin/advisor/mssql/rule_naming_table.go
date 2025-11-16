package mssql

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*NamingTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleTableNaming, &NamingTableAdvisor{})
}

// NamingTableAdvisor is the advisor checking for table naming convention..
type NamingTableAdvisor struct {
}

// Check checks for table naming convention..
func (*NamingTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewNamingTableRule(level, string(checkCtx.Rule.Type), format, maxLength)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// NamingTableRule is the rule for table naming convention.
type NamingTableRule struct {
	BaseRule
	format    *regexp.Regexp
	maxLength int
}

// NewNamingTableRule creates a new NamingTableRule.
func NewNamingTableRule(level storepb.Advice_Status, title string, format *regexp.Regexp, maxLength int) *NamingTableRule {
	return &NamingTableRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		format:    format,
		maxLength: maxLength,
	}
}

// Name returns the rule name.
func (*NamingTableRule) Name() string {
	return "NamingTableRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingTableRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.enterCreateTable(ctx.(*parser.Create_tableContext))
	case "Execute_body":
		r.enterExecuteBody(ctx.(*parser.Execute_bodyContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NamingTableRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *NamingTableRule) enterCreateTable(ctx *parser.Create_tableContext) {
	tableName := ctx.Table_name().GetTable().GetText()

	if !r.format.MatchString(tableName) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf(`%s mismatches table naming convention, naming format should be %q`, tableName, r.format),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf(`%s mismatches table naming convention, its length should be within %d characters`, tableName, r.maxLength),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}

func (r *NamingTableRule) enterExecuteBody(ctx *parser.Execute_bodyContext) {
	if ctx.Func_proc_name_server_database_schema() == nil {
		return
	}
	if ctx.Func_proc_name_server_database_schema().Func_proc_name_database_schema() == nil {
		return
	}
	if ctx.Func_proc_name_server_database_schema().Func_proc_name_database_schema().Func_proc_name_schema() == nil {
		return
	}
	if ctx.Func_proc_name_server_database_schema().Func_proc_name_database_schema().Func_proc_name_schema().GetSchema() != nil {
		return
	}

	v := ctx.Func_proc_name_server_database_schema().Func_proc_name_database_schema().Func_proc_name_schema().GetProcedure()
	_, normalizedProcedureName := tsqlparser.NormalizeTSQLIdentifier(v)
	if normalizedProcedureName != "sp_rename" {
		return
	}

	firstArgument := ctx.Execute_statement_arg()
	if firstArgument == nil {
		return
	}
	if firstArgument.Execute_statement_arg_unnamed() == nil {
		return
	}
	if firstArgument.Execute_statement_arg_unnamed().Execute_parameter() == nil {
		return
	}
	if firstArgument.Execute_statement_arg_unnamed().Execute_parameter().Constant() == nil {
		return
	}
	if firstArgument.Execute_statement_arg_unnamed().Execute_parameter().Constant().STRING() == nil {
		return
	}

	if len(ctx.Execute_statement_arg().AllExecute_statement_arg()) != 1 {
		return
	}
	secondArgument := ctx.Execute_statement_arg().Execute_statement_arg(0)
	if secondArgument == nil {
		return
	}
	if secondArgument.Execute_statement_arg_unnamed() == nil {
		return
	}
	if secondArgument.Execute_statement_arg_unnamed().Execute_parameter() == nil {
		return
	}
	if secondArgument.Execute_statement_arg_unnamed().Execute_parameter().Constant() == nil {
		return
	}
	if secondArgument.Execute_statement_arg_unnamed().Execute_parameter().Constant().STRING() == nil {
		return
	}

	newTableName := secondArgument.Execute_statement_arg_unnamed().Execute_parameter().Constant().STRING().GetText()
	if strings.HasPrefix(newTableName, "'") && strings.HasSuffix(newTableName, "'") {
		newTableName = newTableName[1 : len(newTableName)-1]
	}

	if !r.format.MatchString(newTableName) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf(`%s mismatches table naming convention, naming format should be %q`, newTableName, r.format),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
	if r.maxLength > 0 && len(newTableName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf(`%s mismatches table naming convention, its length should be within %d characters`, newTableName, r.maxLength),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
