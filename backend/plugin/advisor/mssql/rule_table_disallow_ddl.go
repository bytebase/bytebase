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
	_ advisor.Advisor = (*TableDisallowDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DDL, &TableDisallowDDLAdvisor{})
}

// TableDisallowDDLAdvisor is the advisor checking for disallow DDL on specific tables.
type TableDisallowDDLAdvisor struct {
}

func (*TableDisallowDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for table disallow DDL rule")
	}

	// Create the rule
	rule := NewTableDisallowDDLRule(level, checkCtx.Rule.Type.String(), stringArrayPayload.List)

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

// TableDisallowDDLRule is the rule checking for disallow DDL on specific tables.
type TableDisallowDDLRule struct {
	BaseRule
	// disallowList is the list of table names that disallow DDL.
	disallowList []string
}

// NewTableDisallowDDLRule creates a new TableDisallowDDLRule.
func NewTableDisallowDDLRule(level storepb.Advice_Status, title string, disallowList []string) *TableDisallowDDLRule {
	return &TableDisallowDDLRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		disallowList: disallowList,
	}
}

// Name returns the rule name.
func (*TableDisallowDDLRule) Name() string {
	return "TableDisallowDDLRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableDisallowDDLRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.enterCreateTable(ctx.(*parser.Create_tableContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	case NodeTypeDropTable:
		r.enterDropTable(ctx.(*parser.Drop_tableContext))
	case NodeTypeTruncateTable:
		r.enterTruncateTable(ctx.(*parser.Truncate_tableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableDisallowDDLRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *TableDisallowDDLRule) enterCreateTable(ctx *parser.Create_tableContext) {
	tableName := ctx.Table_name()
	if tableName == nil {
		return
	}
	normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "" /* fallbackSchema */, false /* caseSensitive */)
	r.checkTableName(normalizedTableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDDLRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	tableName := ctx.Table_name(0)
	if tableName == nil {
		return
	}
	normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "" /* fallbackSchema */, false /* caseSensitive */)
	r.checkTableName(normalizedTableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDDLRule) enterDropTable(ctx *parser.Drop_tableContext) {
	for _, tableName := range ctx.AllTable_name() {
		if tableName == nil {
			return
		}
		normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "" /* fallbackSchema */, false /* caseSensitive */)
		r.checkTableName(normalizedTableName, ctx.GetStart().GetLine())
	}
}

func (r *TableDisallowDDLRule) enterTruncateTable(ctx *parser.Truncate_tableContext) {
	tableName := ctx.Table_name()
	if tableName == nil {
		return
	}
	normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "" /* fallbackSchema */, false /* caseSensitive */)
	r.checkTableName(normalizedTableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDDLRule) checkTableName(normalizedTableName string, line int) {
	for _, disallow := range r.disallowList {
		if normalizedTableName == disallow {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.TableDisallowDDL.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("DDL is disallowed on table %s.", normalizedTableName),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
			return
		}
	}
}
