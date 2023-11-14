package mysqlwip

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLTableNoFK, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &tableNoFKChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmtNode := range root {
		checker.baseLine = stmtNode.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return checker.adviceList, nil
}

type tableNoFKChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
}

// EnterCreateTable is called when production createTable is entered.
func (checker *tableNoFKChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil || ctx.TableElementList() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement.TableConstraintDef() == nil {
			continue
		}
		checker.handleTableConstraintDef(tableName, tableElement.TableConstraintDef())
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *tableNoFKChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.TableRef() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())

	if ctx.AlterTableActions() == nil || ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}
	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		switch {
		// ADD CONSTRANIT
		case option.ADD_SYMBOL() != nil && option.TableConstraintDef() != nil:
			checker.handleTableConstraintDef(tableName, option.TableConstraintDef())
		default:
			continue
		}
	}
}

func (checker *tableNoFKChecker) handleTableConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) {
	if ctx.GetType_() != nil {
		switch strings.ToUpper(ctx.GetType_().GetText()) {
		case "FOREIGN":
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.TableHasFK,
				Title:   checker.title,
				Content: fmt.Sprintf("Foreign key is not allowed in the table `%s`", tableName),
				Line:    checker.baseLine + ctx.GetStart().GetLine(),
			})
		default:
		}
	}
}
