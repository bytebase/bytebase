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
	_ advisor.Advisor = (*TableDisallowDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleTableDisallowDML, &TableDisallowDMLAdvisor{})
}

// TableDisallowDMLAdvisor is the advisor checking for disallow DML on specific tables.
type TableDisallowDMLAdvisor struct {
}

func (*TableDisallowDMLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	rule := NewTableDisallowDMLRule(level, string(checkCtx.Rule.Type), payload.List)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range list {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// TableDisallowDMLRule checks for disallow DML on specific tables.
type TableDisallowDMLRule struct {
	BaseRule
	disallowList []string
}

// NewTableDisallowDMLRule creates a new TableDisallowDMLRule.
func NewTableDisallowDMLRule(level storepb.Advice_Status, title string, disallowList []string) *TableDisallowDMLRule {
	return &TableDisallowDMLRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		disallowList: disallowList,
	}
}

// Name returns the rule name.
func (*TableDisallowDMLRule) Name() string {
	return "TableDisallowDMLRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableDisallowDMLRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeDeleteStatement:
		r.checkDeleteStatement(ctx.(*mysql.DeleteStatementContext))
	case NodeTypeInsertStatement:
		r.checkInsertStatement(ctx.(*mysql.InsertStatementContext))
	case NodeTypeSelectStatementWithInto:
		r.checkSelectStatementWithInto(ctx.(*mysql.SelectStatementWithIntoContext))
	case NodeTypeUpdateStatement:
		r.checkUpdateStatement(ctx.(*mysql.UpdateStatementContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableDisallowDMLRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableDisallowDMLRule) checkDeleteStatement(ctx *mysql.DeleteStatementContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}
	r.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDMLRule) checkInsertStatement(ctx *mysql.InsertStatementContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}
	r.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDMLRule) checkSelectStatementWithInto(ctx *mysql.SelectStatementWithIntoContext) {
	// Only check text string literal for now.
	if ctx.IntoClause() == nil || ctx.IntoClause().TextStringLiteral() == nil {
		return
	}
	tableName := ctx.IntoClause().TextStringLiteral().GetText()
	r.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (r *TableDisallowDMLRule) checkUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if ctx.TableReferenceList() == nil {
		return
	}
	tables, err := r.extractTableReferenceList(ctx.TableReferenceList())
	if err != nil {
		return
	}
	for _, table := range tables {
		r.checkTableName(table.table, ctx.GetStart().GetLine())
	}
}

func (r *TableDisallowDMLRule) checkTableName(tableName string, line int) {
	for _, disallow := range r.disallowList {
		if tableName == disallow {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.TableDisallowDML.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("DML is disallowed on table %s.", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
			return
		}
	}
}

type table struct {
	database string
	table    string
}

func (r *TableDisallowDMLRule) extractTableReference(ctx mysql.ITableReferenceContext) ([]table, error) {
	if ctx.TableFactor() == nil {
		return nil, nil
	}
	res, err := r.extractTableFactor(ctx.TableFactor())
	if err != nil {
		return nil, err
	}
	for _, joinedTableCtx := range ctx.AllJoinedTable() {
		tables, err := r.extractJoinedTable(joinedTableCtx)
		if err != nil {
			return nil, err
		}
		res = append(res, tables...)
	}

	return res, nil
}

func (*TableDisallowDMLRule) extractTableRef(ctx mysql.ITableRefContext) ([]table, error) {
	if ctx == nil {
		return nil, nil
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx)
	return []table{
		{
			database: databaseName,
			table:    tableName,
		},
	}, nil
}

func (r *TableDisallowDMLRule) extractTableReferenceList(ctx mysql.ITableReferenceListContext) ([]table, error) {
	var res []table
	for _, tableRefCtx := range ctx.AllTableReference() {
		tables, err := r.extractTableReference(tableRefCtx)
		if err != nil {
			return nil, err
		}
		res = append(res, tables...)
	}
	return res, nil
}

func (r *TableDisallowDMLRule) extractTableReferenceListParens(ctx mysql.ITableReferenceListParensContext) ([]table, error) {
	if ctx.TableReferenceList() != nil {
		return r.extractTableReferenceList(ctx.TableReferenceList())
	}
	if ctx.TableReferenceListParens() != nil {
		return r.extractTableReferenceListParens(ctx.TableReferenceListParens())
	}
	return nil, nil
}

func (r *TableDisallowDMLRule) extractTableFactor(ctx mysql.ITableFactorContext) ([]table, error) {
	switch {
	case ctx.SingleTable() != nil:
		return r.extractSingleTable(ctx.SingleTable())
	case ctx.SingleTableParens() != nil:
		return r.extractSingleTableParens(ctx.SingleTableParens())
	case ctx.DerivedTable() != nil:
		return nil, nil
	case ctx.TableReferenceListParens() != nil:
		return r.extractTableReferenceListParens(ctx.TableReferenceListParens())
	case ctx.TableFunction() != nil:
		return nil, nil
	default:
		return nil, nil
	}
}

func (r *TableDisallowDMLRule) extractSingleTable(ctx mysql.ISingleTableContext) ([]table, error) {
	return r.extractTableRef(ctx.TableRef())
}

func (r *TableDisallowDMLRule) extractSingleTableParens(ctx mysql.ISingleTableParensContext) ([]table, error) {
	if ctx.SingleTable() != nil {
		return r.extractSingleTable(ctx.SingleTable())
	}
	if ctx.SingleTableParens() != nil {
		return r.extractSingleTableParens(ctx.SingleTableParens())
	}
	return nil, nil
}

func (r *TableDisallowDMLRule) extractJoinedTable(ctx mysql.IJoinedTableContext) ([]table, error) {
	if ctx.TableFactor() != nil {
		return r.extractTableFactor(ctx.TableFactor())
	}
	if ctx.TableReference() != nil {
		return r.extractTableReference(ctx.TableReference())
	}
	return nil, nil
}
