package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

const (
	innoDB              string = "innodb"
	defaultStorageEngin string = "default_storage_engine"
)

var _ advisor.Advisor = (*UseInnoDBAdvisor)(nil)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleMySQLEngine, &UseInnoDBAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleMySQLEngine, &UseInnoDBAdvisor{})
}

// UseInnoDBAdvisor is the advisor checking for using InnoDB engine.
type UseInnoDBAdvisor struct {
}

// Check checks for using InnoDB engine.
func (*UseInnoDBAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	list, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewUseInnoDBRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range list {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// UseInnoDBRule checks for using InnoDB engine.
type UseInnoDBRule struct {
	BaseRule
}

// NewUseInnoDBRule creates a new UseInnoDBRule.
func NewUseInnoDBRule(level storepb.Advice_Status, title string) *UseInnoDBRule {
	return &UseInnoDBRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*UseInnoDBRule) Name() string {
	return "UseInnoDBRule"
}

// OnEnter is called when entering a parse tree node.
func (r *UseInnoDBRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	case NodeTypeSetStatement:
		r.checkSetStatement(ctx.(*mysql.SetStatementContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*UseInnoDBRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *UseInnoDBRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.CreateTableOptions() == nil {
		return
	}
	for _, tableOption := range ctx.CreateTableOptions().AllCreateTableOption() {
		if tableOption.ENGINE_SYMBOL() != nil && tableOption.EngineRef() != nil {
			if tableOption.EngineRef().TextOrIdentifier() == nil {
				continue
			}
			engine := mysqlparser.NormalizeMySQLTextOrIdentifier(tableOption.EngineRef().TextOrIdentifier())
			if strings.ToLower(engine) != innoDB {
				content := "CREATE " + ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
				line := tableOption.GetStart().GetLine()
				r.addAdvice(content, line)
				break
			}
		}
	}
}

func (r *UseInnoDBRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.AlterTableActions() == nil || ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}
	code := advisorcode.Ok
	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllCreateTableOptionsSpaceSeparated() {
		for _, op := range option.AllCreateTableOption() {
			if op.ENGINE_SYMBOL() != nil {
				if op.EngineRef() == nil {
					continue
				}
				engine := op.EngineRef().GetText()
				if strings.ToLower(engine) != innoDB {
					code = advisorcode.NotInnoDBEngine
					break
				}
			}
		}
	}

	if code != advisorcode.Ok {
		content := "ALTER " + ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
		line := ctx.GetStart().GetLine()
		r.addAdvice(content, line)
	}
}

func (r *UseInnoDBRule) checkSetStatement(ctx *mysql.SetStatementContext) {
	code := advisorcode.Ok
	if ctx.StartOptionValueList() == nil {
		return
	}

	startOptionValueList := ctx.StartOptionValueList()
	if startOptionValueList.OptionValueNoOptionType() == nil {
		return
	}
	optionValueNoOptionType := startOptionValueList.OptionValueNoOptionType()
	if optionValueNoOptionType.InternalVariableName() == nil {
		return
	}
	name := optionValueNoOptionType.InternalVariableName().GetText()
	if strings.ToLower(name) != defaultStorageEngin {
		return
	}
	if optionValueNoOptionType.SetExprOrDefault() != nil {
		engine := optionValueNoOptionType.SetExprOrDefault().GetText()
		if strings.ToLower(engine) != innoDB {
			code = advisorcode.NotInnoDBEngine
		}
	}

	if code != advisorcode.Ok {
		content := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
		line := ctx.GetStart().GetLine()
		r.addAdvice(content, line)
	}
}

func (r *UseInnoDBRule) addAdvice(content string, lineNumber int) {
	lineNumber += r.baseLine
	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          advisorcode.NotInnoDBEngine.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("\"%s;\" doesn't use InnoDB engine", content),
		StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
	})
}
