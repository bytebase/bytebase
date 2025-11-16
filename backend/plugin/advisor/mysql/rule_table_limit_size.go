package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleTableLimitSize, &MaximumTableSizeAdvisor{})
}

type MaximumTableSizeAdvisor struct {
}

var (
	_ advisor.Advisor = &MaximumTableSizeAdvisor{}
)

// If table size > xx bytes, then warning/error.
func (*MaximumTableSizeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	// User defined rule level.
	status, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice

	for _, parsedResult := range stmtList {
		statTypeChecker := &mysqlparser.StatementTypeChecker{}
		antlr.ParseTreeWalkerDefault.Walk(statTypeChecker, parsedResult.Tree)

		if statTypeChecker.IsDDL {
			// Create the rule
			rule := NewTableLimitSizeRule(status, string(checkCtx.Rule.Type), payload.Number, checkCtx.DBSchema)

			// Create the generic checker with the rule
			checker := NewGenericChecker([]Rule{rule})

			rule.SetBaseLine(parsedResult.BaseLine)
			checker.SetBaseLine(parsedResult.BaseLine)
			antlr.ParseTreeWalkerDefault.Walk(checker, parsedResult.Tree)

			// Generate advice based on collected table information
			rule.generateAdvice()

			adviceList = append(adviceList, checker.GetAdviceList()...)
		}
	}

	return adviceList, nil
}

// TableLimitSizeRule checks for table size limits.
type TableLimitSizeRule struct {
	BaseRule
	affectedTabNames  []string
	maxRows           int
	dbSchema          *storepb.DatabaseSchemaMetadata
	statementBaseLine int
}

// NewTableLimitSizeRule creates a new TableLimitSizeRule.
func NewTableLimitSizeRule(level storepb.Advice_Status, title string, maxRows int, dbSchema *storepb.DatabaseSchemaMetadata) *TableLimitSizeRule {
	return &TableLimitSizeRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		maxRows:  maxRows,
		dbSchema: dbSchema,
	}
}

// Name returns the rule name.
func (*TableLimitSizeRule) Name() string {
	return "TableLimitSizeRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableLimitSizeRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	case NodeTypeTruncateTableStatement:
		r.checkTruncateTableStatement(ctx.(*mysql.TruncateTableStatementContext))
	case NodeTypeDropTable:
		r.checkDropTable(ctx.(*mysql.DropTableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableLimitSizeRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableLimitSizeRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	r.statementBaseLine = ctx.GetStart().GetLine()
	r.affectedTabNames = append(r.affectedTabNames, tableName)
}

func (r *TableLimitSizeRule) checkTruncateTableStatement(ctx *mysql.TruncateTableStatementContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	r.statementBaseLine = ctx.GetStart().GetLine()
	r.affectedTabNames = append(r.affectedTabNames, tableName)
}

func (r *TableLimitSizeRule) checkDropTable(ctx *mysql.DropTableContext) {
	r.statementBaseLine = ctx.GetStart().GetLine()
	for _, tabRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(tabRef)
		r.affectedTabNames = append(r.affectedTabNames, tableName)
	}
}

func (r *TableLimitSizeRule) generateAdvice() {
	if r.dbSchema != nil && len(r.dbSchema.Schemas) != 0 {
		// Check all table size.
		for _, tabName := range r.affectedTabNames {
			tableRows := getTabRowsByName(tabName, r.dbSchema.Schemas[0].Tables)
			if tableRows >= int64(r.maxRows) {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          code.TableExceedLimitSize.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("Apply DDL on large table '%s' ( %d rows ) will lock table for a long time", tabName, tableRows),
					StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + r.statementBaseLine),
				})
			}
		}
	}
}

func getTabRowsByName(targetTabName string, tables []*storepb.TableMetadata) int64 {
	for _, table := range tables {
		if table.Name == targetTabName {
			return table.RowCount
		}
	}
	return 0
}
