package pg

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementCreateSpecifySchema)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_CREATE_SPECIFY_SCHEMA, &StatementCreateSpecifySchema{})
}

type StatementCreateSpecifySchema struct {
}

func (*StatementCreateSpecifySchema) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	parseResults, err := advisor.GetANTLRParseResults(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementCreateSpecifySchemaRule{
		BaseRule: BaseRule{
			level: level,
			title: checkCtx.Rule.Type.String(),
		},
	}

	checker := NewGenericChecker([]Rule{rule})

	for _, parseResult := range parseResults {
		rule.SetBaseLine(parseResult.BaseLine)
		checker.SetBaseLine(parseResult.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, parseResult.Tree)
	}

	return checker.GetAdviceList(), nil
}

type statementCreateSpecifySchemaRule struct {
	BaseRule
}

func (*statementCreateSpecifySchemaRule) Name() string {
	return "statement_create_specify_schema"
}

func (r *statementCreateSpecifySchemaRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx)
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*statementCreateSpecifySchemaRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt handles CREATE TABLE
func (r *statementCreateSpecifySchemaRule) handleCreatestmt(ctx antlr.ParserRuleContext) {
	createstmtCtx, ok := ctx.(*parser.CreatestmtContext)
	if !ok {
		return
	}

	if !isTopLevel(createstmtCtx.GetParent()) {
		return
	}

	allQualifiedNames := createstmtCtx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		schemaName := extractSchemaName(allQualifiedNames[0])
		if schemaName == "" {
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    code.StatementCreateWithoutSchemaName.Int32(),
				Title:   r.title,
				Content: "Table schema should be specified.",
				StartPosition: &storepb.Position{
					Line:   int32(createstmtCtx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}
