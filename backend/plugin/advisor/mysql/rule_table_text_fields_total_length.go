package mysql

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleTableTextFieldsTotalLength, &TableMaximumVarcharLengthAdvisor{})
}

type TableMaximumVarcharLengthAdvisor struct {
}

func (*TableMaximumVarcharLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewTableTextFieldsTotalLengthRule(level, string(checkCtx.Rule.Type), checkCtx.FinalCatalog, payload.Number)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// TableTextFieldsTotalLengthRule checks for table text fields total length.
type TableTextFieldsTotalLengthRule struct {
	BaseRule
	finalCatalog *catalog.DatabaseState
	maximum      int
}

// NewTableTextFieldsTotalLengthRule creates a new TableTextFieldsTotalLengthRule.
func NewTableTextFieldsTotalLengthRule(level storepb.Advice_Status, title string, finalCatalog *catalog.DatabaseState, maximum int) *TableTextFieldsTotalLengthRule {
	return &TableTextFieldsTotalLengthRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		finalCatalog: finalCatalog,
		maximum:      maximum,
	}
}

// Name returns the rule name.
func (*TableTextFieldsTotalLengthRule) Name() string {
	return "TableTextFieldsTotalLengthRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableTextFieldsTotalLengthRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableTextFieldsTotalLengthRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableTextFieldsTotalLengthRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableElementList() == nil || ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if tableName == "" {
		return
	}
	tableInfo := r.finalCatalog.GetTable("", tableName)
	if tableInfo == nil {
		return
	}
	total := getTotalTextLength(tableInfo)
	if total > int64(r.maximum) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.IndexCountExceedsLimit.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Table %q total text column length (%d) exceeds the limit (%d).", tableName, total, r.maximum),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}

func (r *TableTextFieldsTotalLengthRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
	tableInfo := r.finalCatalog.GetTable("", tableName)
	if tableInfo == nil {
		return
	}
	total := getTotalTextLength(tableInfo)
	if total > int64(r.maximum) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.TotalTextLengthExceedsLimit.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Table %q total text column length (%d) exceeds the limit (%d).", tableName, total, r.maximum),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}

func getTotalTextLength(tableInfo *catalog.TableState) int64 {
	var total int64
	columns := tableInfo.ListColumns()
	for _, column := range columns {
		total += getTextLength(column.Type())
	}
	return total
}

func getTextLength(s string) int64 {
	s = strings.ToLower(s)
	switch s {
	case "char", "binary":
		return 1
	case "tinyblob", "tinytext":
		return 255
	case "blob", "text":
		return 65_535
	case "mediumblob", "mediumtext":
		return 16_777_215
	case "longblob", "longtext":
		return 4_294_967_295
	default:
		re := regexp.MustCompile(`[a-z]+\((\d+)\)`)
		match := re.FindStringSubmatch(s)
		if len(match) >= 2 {
			n, err := strconv.ParseInt(match[1], 10, 64)
			if err == nil {
				return int64(n)
			}
		}
	}
	return 0
}
