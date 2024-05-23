package mssql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	advisor.Register(store.Engine_MSSQL, advisor.MSSQLFunctionDisallowCreateOrAlter, &FunctionDisallowCreateOrAlterAdvisor{})
}

type FunctionDisallowCreateOrAlterAdvisor struct{}

type FunctionDisallowCreateOrAlterChecker struct {
	*parser.BaseTSqlParserListener

	level      advisor.Status
	title      string
	adviceList []advisor.Advice
}

// Check implements advisor.Advisor.
func (*FunctionDisallowCreateOrAlterAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &FunctionDisallowCreateOrAlterChecker{
		level: level,
		title: ctx.Rule.Type,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

func (checker *FunctionDisallowCreateOrAlterChecker) EnterCreate_or_alter_function(ctx *parser.Create_or_alter_functionContext) {
	checker.adviceList = append(checker.adviceList, advisor.Advice{
		Status:  checker.level,
		Code:    advisor.DisallowCreateFunction,
		Title:   checker.title,
		Content: "Creating or altering functions is prohibited",
		Line:    ctx.GetStart().GetLine(),
	})
}
