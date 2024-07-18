package mssql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.MSSQLProcedureDisallowCreateOrAlter, &FunctionDisallowCreateOrAlterAdvisor{})
}

type ProcedureDisallowCreateOrAlterAdvisor struct{}

type ProcedureDisallowCreateOrAlterChecker struct {
	*parser.BaseTSqlParserListener

	level      storepb.Advice_Status
	title      string
	adviceList []*storepb.Advice
}

func (*ProcedureDisallowCreateOrAlterAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &ProcedureDisallowCreateOrAlterChecker{
		level: level,
		title: ctx.Rule.Type,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.adviceList, nil
}

func (checker *FunctionDisallowCreateOrAlterChecker) EnterCreate_or_alter_procedure(ctx *parser.Create_or_alter_procedureContext) {
	checker.adviceList = append(checker.adviceList, &storepb.Advice{
		Status:  checker.level,
		Code:    advisor.DisallowCreateFunction.Int32(),
		Title:   checker.title,
		Content: "Creating or altering procedures is prohibited",
		StartPosition: &storepb.Position{
			Line: int32(ctx.GetStart().GetLine()),
		},
	})
}
