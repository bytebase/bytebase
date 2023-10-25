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

const (
	innoDB              string = "innodb"
	defaultStorageEngin string = "default_storage_engine"
)

var _ advisor.Advisor = (*UseInnoDBAdvisor)(nil)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLUseInnoDB, &UseInnoDBAdvisor{})
}

// UseInnoDBAdvisor is the advisor checking for using InnoDB engine.
type UseInnoDBAdvisor struct {
}

// Check checks for using InnoDB engine.
func (*UseInnoDBAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	list, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &useInnoDBChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range list {
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
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

type useInnoDBChecker struct {
	*mysql.BaseMySQLParserListener

	adviceList []advisor.Advice
	level      advisor.Status
	title      string
}

// EnterCreateTable is called when production createTable is entered.
func (c *useInnoDBChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.CreateTableOptions() == nil {
		return
	}
	code := advisor.Ok
	for _, tableOption := range ctx.CreateTableOptions().AllCreateTableOption() {
		if tableOption.ENGINE_SYMBOL() != nil && tableOption.EngineRef() != nil {
			engine := mysqlparser.NormalizeMySQLTextOrIdentifier(tableOption.EngineRef().TextOrIdentifier())
			if strings.ToLower(engine) != innoDB {
				code = advisor.NotInnoDBEngine
				break
			}
		}
	}

	if code != advisor.Ok {
		c.adviceList = append(c.adviceList, advisor.Advice{
			Status:  c.level,
			Code:    code,
			Title:   c.title,
			Content: fmt.Sprintf("\"CREATE %s;\" doesn't use InnoDB engine", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)),
			Line:    ctx.GetStart().GetLine(),
		})
	}
}

func (c *useInnoDBChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	code := advisor.Ok
	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllCreateTableOptionsSpaceSeparated() {
		for _, op := range option.AllCreateTableOption() {
			switch {
			case op.ENGINE_SYMBOL() != nil:
				engine := op.EngineRef().GetText()
				if strings.ToLower(engine) != innoDB {
					code = advisor.NotInnoDBEngine
					break
				}
			default:
			}
		}
	}

	if code != advisor.Ok {
		c.adviceList = append(c.adviceList, advisor.Advice{
			Status:  c.level,
			Code:    code,
			Title:   c.title,
			Content: fmt.Sprintf("\"ALTER %s;\" doesn't use InnoDB engine", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)),
			Line:    ctx.GetStart().GetLine(),
		})
	}
}

func (c *useInnoDBChecker) EnterSetStatement(ctx *mysql.SetStatementContext) {
	code := advisor.Ok
	if ctx.StartOptionValueList() == nil {
		return
	}

	startOptionValueList := ctx.StartOptionValueList()
	if startOptionValueList.OptionValueNoOptionType() == nil {
		return
	}
	optionValueNoOptionType := startOptionValueList.OptionValueNoOptionType()
	if optionValueNoOptionType.SetExprOrDefault() != nil {
		engine := optionValueNoOptionType.SetExprOrDefault().GetText()
		if strings.ToLower(engine) != innoDB {
			code = advisor.NotInnoDBEngine
		}
	}

	if code != advisor.Ok {
		c.adviceList = append(c.adviceList, advisor.Advice{
			Status:  c.level,
			Code:    code,
			Title:   c.title,
			Content: fmt.Sprintf("\"%s;\" doesn't use InnoDB engine", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)),
			Line:    ctx.GetStart().GetLine(),
		})
	}
}
