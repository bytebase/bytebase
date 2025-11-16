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
	_ advisor.Advisor = (*TableDisallowDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleTableDisallowDDL, &TableDisallowDDLAdvisor{})
}

// TableDisallowDDLAdvisor is the advisor checking for disallow DDL on specific tables.
type TableDisallowDDLAdvisor struct {
}

func (*TableDisallowDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	list, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
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
	rule := NewTableDisallowDDLRule(level, string(checkCtx.Rule.Type), payload.List)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range list {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// TableDisallowDDLRule checks for disallow DDL on specific tables.
type TableDisallowDDLRule struct {
	BaseRule
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
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeDropTable:
		r.checkDropTable(ctx.(*mysql.DropTableContext))
	case NodeTypeRenameTableStatement:
		r.checkRenameTableStatement(ctx.(*mysql.RenameTableStatementContext))
	case NodeTypeTruncateTableStatement:
		r.checkTruncateTableStatement(ctx.(*mysql.TruncateTableStatementContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableDisallowDDLRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableDisallowDDLRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}
	r.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDDLRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if tableName == "" {
		return
	}
	r.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDDLRule) checkDropTable(ctx *mysql.DropTableContext) {
	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(tableRef)
		if tableName == "" {
			continue
		}
		r.checkTableName(tableName, ctx.GetStart().GetLine())
	}
}

func (r *TableDisallowDDLRule) checkRenameTableStatement(ctx *mysql.RenameTableStatementContext) {
	for _, renamePair := range ctx.AllRenamePair() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(renamePair.TableRef())
		if tableName == "" {
			continue
		}
		r.checkTableName(tableName, ctx.GetStart().GetLine())
	}
}

func (r *TableDisallowDDLRule) checkTruncateTableStatement(ctx *mysql.TruncateTableStatementContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}
	r.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDDLRule) checkTableName(tableName string, line int) {
	for _, disallow := range r.disallowList {
		if tableName == disallow {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.TableDisallowDDL.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("DDL is disallowed on table %s.", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
			return
		}
	}
}
