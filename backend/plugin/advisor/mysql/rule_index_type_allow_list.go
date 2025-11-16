package mysql

import (
	"context"
	"fmt"
	"slices"

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
	_ advisor.Advisor = (*IndexTypeAllowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleIndexTypeAllowList, &IndexTypeAllowListAdvisor{})
}

// IndexTypeAllowListAdvisor is the advisor checking for index types.
type IndexTypeAllowListAdvisor struct {
}

func (*IndexTypeAllowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewIndexTypeAllowListRule(level, string(checkCtx.Rule.Type), payload.List)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// IndexTypeAllowListRule checks for index types.
type IndexTypeAllowListRule struct {
	BaseRule
	allowList []string
}

// NewIndexTypeAllowListRule creates a new IndexTypeAllowListRule.
func NewIndexTypeAllowListRule(level storepb.Advice_Status, title string, allowList []string) *IndexTypeAllowListRule {
	return &IndexTypeAllowListRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		allowList: allowList,
	}
}

// Name returns the rule name.
func (*IndexTypeAllowListRule) Name() string {
	return "IndexTypeAllowListRule"
}

// OnEnter is called when entering a parse tree node.
func (r *IndexTypeAllowListRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	case NodeTypeCreateIndex:
		r.checkCreateIndex(ctx.(*mysql.CreateIndexContext))
	default:
		// No action required for other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*IndexTypeAllowListRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *IndexTypeAllowListRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}
	if ctx.TableElementList() == nil {
		return
	}

	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement == nil || tableElement.TableConstraintDef() == nil {
			continue
		}
		r.handleConstraintDef(tableElement.TableConstraintDef())
	}
}

func (r *IndexTypeAllowListRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRef() == nil {
		return
	}
	if ctx.AlterTableActions() == nil || ctx.AlterTableActions().AlterCommandList() == nil || ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	for _, alterListItem := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if alterListItem == nil {
			continue
		}
		if alterListItem.ADD_SYMBOL() != nil && alterListItem.TableConstraintDef() != nil {
			r.handleConstraintDef(alterListItem.TableConstraintDef())
		}
	}
}

func (r *IndexTypeAllowListRule) handleConstraintDef(ctx mysql.ITableConstraintDefContext) {
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserINDEX_SYMBOL, mysql.MySQLParserKEY_SYMBOL, mysql.MySQLParserPRIMARY_SYMBOL, mysql.MySQLParserUNIQUE_SYMBOL, mysql.MySQLParserFULLTEXT_SYMBOL, mysql.MySQLParserSPATIAL_SYMBOL:
	default:
		return
	}

	indexType := "BTREE"
	if ctx.IndexNameAndType() != nil && ctx.IndexNameAndType().IndexType() != nil {
		indexType = ctx.IndexNameAndType().IndexType().GetText()
	} else {
		if ctx.FULLTEXT_SYMBOL() != nil {
			indexType = "FULLTEXT"
		} else if ctx.SPATIAL_SYMBOL() != nil {
			indexType = "SPATIAL"
		}
	}
	r.validateIndexType(indexType, ctx.GetStart().GetLine())
}

func (r *IndexTypeAllowListRule) checkCreateIndex(ctx *mysql.CreateIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil || ctx.CreateIndexTarget().KeyListVariants() == nil {
		return
	}

	indexType := "BTREE"
	if ctx.IndexNameAndType() != nil && ctx.IndexNameAndType().IndexType() != nil {
		indexType = ctx.IndexNameAndType().IndexType().GetText()
	} else {
		if ctx.FULLTEXT_SYMBOL() != nil {
			indexType = "FULLTEXT"
		} else if ctx.SPATIAL_SYMBOL() != nil {
			indexType = "SPATIAL"
		}
	}
	r.validateIndexType(indexType, ctx.GetStart().GetLine())
}

// validateIndexType checks if the index type is in the allow list.
func (r *IndexTypeAllowListRule) validateIndexType(indexType string, line int) {
	if slices.Contains(r.allowList, indexType) {
		return
	}

	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          code.IndexTypeNotAllowed.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("Index type `%s` is not allowed", indexType),
		StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + line),
	})
}
