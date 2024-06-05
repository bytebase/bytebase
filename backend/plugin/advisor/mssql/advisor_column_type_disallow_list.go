// Package mssql is the advisor for MSSQL database.
package mssql

import (
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.MSSQLColumnTypeDisallowList, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for disallowed types for column.
type ColumnTypeDisallowListAdvisor struct {
}

func (*ColumnTypeDisallowListAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &columnTypeDisallowListChecker{
		level:         level,
		title:         string(ctx.Rule.Type),
		disallowTypes: []string{},
	}
	for _, tp := range payload.List {
		checker.disallowTypes = append(checker.disallowTypes, strings.ToUpper(tp))
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:  storepb.Advice_SUCCESS,
			Code:    advisor.Ok.Int32(),
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

// columnTypeDisallowListChecker is the checker for column disallow types.
type columnTypeDisallowListChecker struct {
	*parser.BaseTSqlParserListener

	level         storepb.Advice_Status
	title         string
	adviceList    []*storepb.Advice
	disallowTypes []string
}

func (checker *columnTypeDisallowListChecker) EnterData_type(ctx *parser.Data_typeContext) {
	formatedDataType := strings.ToUpper(ctx.GetText())
	if slices.Contains(checker.disallowTypes, formatedDataType) {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:  checker.level,
			Code:    advisor.DisabledColumnType.Int32(),
			Title:   checker.title,
			Content: "Column type " + formatedDataType + " is disallowed",
			StartPosition: &storepb.Position{
				Line: int32(ctx.GetStart().GetLine()),
			},
		})
	}
}
