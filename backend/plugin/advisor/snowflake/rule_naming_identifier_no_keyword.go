// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*NamingIdentifierNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleIdentifierNoKeyword, &NamingIdentifierNoKeywordAdvisor{})
}

// NamingIdentifierNoKeywordAdvisor is the advisor checking for identifier naming convention without keyword.
type NamingIdentifierNoKeywordAdvisor struct {
}

// Check checks for identifier naming convention without keyword.
func (*NamingIdentifierNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewNamingIdentifierNoKeywordRule(level, string(checkCtx.Rule.Type))
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// NamingIdentifierNoKeywordRule checks for identifier naming without keywords.
type NamingIdentifierNoKeywordRule struct {
	BaseRule
	// currentOriginalTableName is the original table name in the statement.
	currentOriginalTableName string
}

// NewNamingIdentifierNoKeywordRule creates a new NamingIdentifierNoKeywordRule.
func NewNamingIdentifierNoKeywordRule(level storepb.Advice_Status, title string) *NamingIdentifierNoKeywordRule {
	return &NamingIdentifierNoKeywordRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*NamingIdentifierNoKeywordRule) Name() string {
	return "NamingIdentifierNoKeywordRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingIdentifierNoKeywordRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.currentOriginalTableName = ctx.(*parser.Create_tableContext).Object_name().GetText()
	case "Create_table_as_select":
		r.currentOriginalTableName = ctx.(*parser.Create_table_as_selectContext).Object_name().GetText()
	case NodeTypeColumnDeclItemList:
		r.enterColumnDeclItemList(ctx.(*parser.Column_decl_item_listContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *NamingIdentifierNoKeywordRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable, "Create_table_as_select":
		r.currentOriginalTableName = ""
	default:
		// Ignore other node types
	}
	return nil
}

func (r *NamingIdentifierNoKeywordRule) enterColumnDeclItemList(ctx *parser.Column_decl_item_listContext) {
	if r.currentOriginalTableName == "" {
		return
	}

	allItems := ctx.AllColumn_decl_item()
	if len(allItems) == 0 {
		return
	}

	for _, item := range allItems {
		if fullColDecl := item.Full_col_decl(); fullColDecl != nil {
			originalID := fullColDecl.Col_decl().Column_name().Id_()
			originalColName := snowsqlparser.NormalizeSnowSQLObjectNamePart(originalID)
			if snowsqlparser.IsSnowflakeKeyword(originalColName, false) {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          code.NameIsKeywordIdentifier.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("Identifier %s is a keyword and should be avoided", originalID.GetText()),
					StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
				})
			}
		}
	}
}

func (r *NamingIdentifierNoKeywordRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	if ctx.Table_column_action() == nil || ctx.Table_column_action().RENAME() == nil {
		return
	}
	r.currentOriginalTableName = ctx.Object_name(0).GetText()
	renameToID := ctx.Table_column_action().Column_name(1).Id_()
	renameToColName := snowsqlparser.NormalizeSnowSQLObjectNamePart(renameToID)
	if snowsqlparser.IsSnowflakeKeyword(renameToColName, false) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NameIsKeywordIdentifier.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Identifier %s is a keyword and should be avoided", renameToID.GetText()),
			StartPosition: common.ConvertANTLRLineToPosition(renameToID.GetStart().GetLine()),
		})
	}
}
