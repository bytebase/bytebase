package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnDisallowSetCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnDisallowSetCharset, &ColumnDisallowSetCharsetAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnDisallowSetCharset, &ColumnDisallowSetCharsetAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnDisallowSetCharset, &ColumnDisallowSetCharsetAdvisor{})
}

// ColumnDisallowSetCharsetAdvisor is the advisor checking for disallow set column charset.
type ColumnDisallowSetCharsetAdvisor struct {
}

// Check checks for disallow set column charset.
func (*ColumnDisallowSetCharsetAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnDisallowSetCharsetRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnDisallowSetCharsetRule checks for disallow set column charset.
type ColumnDisallowSetCharsetRule struct {
	BaseRule
	text string
}

// NewColumnDisallowSetCharsetRule creates a new ColumnDisallowSetCharsetRule.
func NewColumnDisallowSetCharsetRule(level storepb.Advice_Status, title string) *ColumnDisallowSetCharsetRule {
	return &ColumnDisallowSetCharsetRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ColumnDisallowSetCharsetRule) Name() string {
	return "ColumnDisallowSetCharsetRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnDisallowSetCharsetRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		r.checkQuery(ctx.(*mysql.QueryContext))
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
		// Other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ColumnDisallowSetCharsetRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnDisallowSetCharsetRule) checkQuery(ctx *mysql.QueryContext) {
	r.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (r *ColumnDisallowSetCharsetRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableElementList() == nil || ctx.TableName() == nil {
		return
	}

	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement.ColumnDefinition() == nil {
			continue
		}
		if tableElement.ColumnDefinition().FieldDefinition() == nil {
			continue
		}
		if tableElement.ColumnDefinition().FieldDefinition().DataType() == nil {
			continue
		}
		charset := r.getCharSet(tableElement.ColumnDefinition().FieldDefinition().DataType())
		if !r.checkCharset(charset) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.SetColumnCharset.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Disallow set column charset but \"%s\" does", r.text),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		}
	}
}

func (r *ColumnDisallowSetCharsetRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}
	// alter table add column, change column, modify column.
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		var charsetList []string
		switch {
		// add column.
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				if item.FieldDefinition().DataType() == nil {
					continue
				}

				charsetName := r.getCharSet(item.FieldDefinition().DataType())
				charsetList = append(charsetList, charsetName)
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil {
						continue
					}
					if tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					if tableElement.ColumnDefinition().FieldDefinition().DataType() == nil {
						continue
					}

					charsetName := r.getCharSet(tableElement.ColumnDefinition().FieldDefinition().DataType())
					charsetList = append(charsetList, charsetName)
				}
			default:
				// Other add column formats
			}
		// change column.
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			charsetName := r.getCharSet(item.FieldDefinition().DataType())
			charsetList = append(charsetList, charsetName)
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			charsetName := r.getCharSet(item.FieldDefinition().DataType())
			charsetList = append(charsetList, charsetName)
		default:
			continue
		}

		for _, charsetName := range charsetList {
			if !r.checkCharset(charsetName) {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          code.SetColumnCharset.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("Disallow set column charset but \"%s\" does", r.text),
					StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
				})
			}
		}
	}
}

func (*ColumnDisallowSetCharsetRule) getCharSet(ctx mysql.IDataTypeContext) string {
	if ctx.CharsetWithOptBinary() == nil {
		return ""
	}
	charset := mysqlparser.NormalizeMySQLCharsetName(ctx.CharsetWithOptBinary().CharsetName())
	return charset
}

func (*ColumnDisallowSetCharsetRule) checkCharset(charset string) bool {
	switch charset {
	// empty charset or binary for JSON.
	case "", "binary":
		return true
	default:
		return false
	}
}
