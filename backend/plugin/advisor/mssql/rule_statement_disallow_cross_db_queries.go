package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleStatementDisallowCrossDBQueries, &DisallowCrossDBQueriesAdvisor{})
}

type DisallowCrossDBQueriesAdvisor struct{}

func (*DisallowCrossDBQueriesAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewDisallowCrossDBQueriesRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// DisallowCrossDBQueriesRule is the rule for disallowing cross database queries.
type DisallowCrossDBQueriesRule struct {
	BaseRule
	curDB string
}

// NewDisallowCrossDBQueriesRule creates a new DisallowCrossDBQueriesRule.
func NewDisallowCrossDBQueriesRule(level storepb.Advice_Status, title string, currentDatabase string) *DisallowCrossDBQueriesRule {
	return &DisallowCrossDBQueriesRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		curDB: currentDatabase,
	}
}

// Name returns the rule name.
func (*DisallowCrossDBQueriesRule) Name() string {
	return "DisallowCrossDBQueriesRule"
}

// OnEnter is called when entering a parse tree node.
func (r *DisallowCrossDBQueriesRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Table_source_item":
		r.enterTableSourceItem(ctx.(*parser.Table_source_itemContext))
	case "Use_statement":
		r.enterUseStatement(ctx.(*parser.Use_statementContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*DisallowCrossDBQueriesRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *DisallowCrossDBQueriesRule) enterTableSourceItem(ctx *parser.Table_source_itemContext) {
	if fullTblnameCtx := ctx.Full_table_name(); fullTblnameCtx != nil {
		// Case insensitive.
		if fullTblName, err := tsql.NormalizeFullTableName(fullTblnameCtx); err == nil && fullTblName.Database != "" && !strings.EqualFold(fullTblName.Database, r.curDB) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.StatementDisallowCrossDBQueries.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Cross database queries (target databse: '%s', current database: '%s') are prohibited", fullTblName.Database, r.curDB),
				StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
			})
		}
		// Ignore internal error...
	}
}

func (r *DisallowCrossDBQueriesRule) enterUseStatement(ctx *parser.Use_statementContext) {
	if newDB := ctx.GetDatabase(); newDB != nil {
		_, lowercaceDBName := tsql.NormalizeTSQLIdentifier(newDB)
		r.curDB = lowercaceDBName
	}
}
