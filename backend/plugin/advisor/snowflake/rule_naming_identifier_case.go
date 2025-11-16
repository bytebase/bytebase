// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*NamingIdentifierCaseAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleIdentifierCase, &NamingIdentifierCaseAdvisor{})
}

// NamingIdentifierCaseAdvisor is the advisor checking for identifier case.
type NamingIdentifierCaseAdvisor struct {
}

// Check checks for identifier case.
func (*NamingIdentifierCaseAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewNamingIdentifierCaseRule(level, string(checkCtx.Rule.Type))
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// NamingIdentifierCaseRule checks for identifier case.
type NamingIdentifierCaseRule struct {
	BaseRule
	// currentOriginalTableName is the original table name in the statement.
	currentOriginalTableName string
}

// NewNamingIdentifierCaseRule creates a new NamingIdentifierCaseRule.
func NewNamingIdentifierCaseRule(level storepb.Advice_Status, title string) *NamingIdentifierCaseRule {
	return &NamingIdentifierCaseRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*NamingIdentifierCaseRule) Name() string {
	return "NamingIdentifierCaseRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingIdentifierCaseRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
		// Other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *NamingIdentifierCaseRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable, "Create_table_as_select":
		r.currentOriginalTableName = ""
	default:
		// Other node types
	}
	return nil
}

func (r *NamingIdentifierCaseRule) enterColumnDeclItemList(ctx *parser.Column_decl_item_listContext) {
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
			if strings.ToUpper(originalColName) != originalColName {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          code.NamingCaseMismatch.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("Identifier %q should be upper case", originalColName),
					StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
				})
			}
		}
	}
}

func (r *NamingIdentifierCaseRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	if ctx.Table_column_action() == nil || ctx.Table_column_action().RENAME() == nil {
		return
	}
	r.currentOriginalTableName = ctx.Object_name(0).GetText()
	renameToID := ctx.Table_column_action().Column_name(1).Id_()
	renameToColName := snowsqlparser.NormalizeSnowSQLObjectNamePart(renameToID)
	if strings.ToUpper(renameToColName) != renameToColName {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingCaseMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Identifier %q should be upper case", renameToColName),
			StartPosition: common.ConvertANTLRLineToPosition(renameToID.GetStart().GetLine()),
		})
	}
}
