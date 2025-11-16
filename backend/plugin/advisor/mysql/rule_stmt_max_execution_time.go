package mysql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*MaxExecutionTimeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementMaxExecutionTime, &MaxExecutionTimeAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementMaxExecutionTime, &MaxExecutionTimeAdvisor{})
}

// MaxExecutionTimeAdvisor is the advisor checking for the max execution time.
type MaxExecutionTimeAdvisor struct {
}

func (*MaxExecutionTimeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to stmt list")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	systemVariable := "max_execution_time"
	// For MariaDB, the system variable is `max_statement_time`.
	if checkCtx.Rule.Engine == storepb.Engine_MARIADB {
		systemVariable = "max_statement_time"
	}

	// Create the rule
	rule := NewMaxExecutionTimeRule(level, string(checkCtx.Rule.Type), systemVariable)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// MaxExecutionTimeRule checks for the max execution time.
type MaxExecutionTimeRule struct {
	BaseRule
	// The system variable name for max execution time.
	// For MySQL, it is `max_execution_time`.
	// For MariaDB, it is `max_statement_time`.
	systemVariable string
	hasSet         bool
}

// NewMaxExecutionTimeRule creates a new MaxExecutionTimeRule.
func NewMaxExecutionTimeRule(level storepb.Advice_Status, title string, systemVariable string) *MaxExecutionTimeRule {
	return &MaxExecutionTimeRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		systemVariable: systemVariable,
	}
}

// Name returns the rule name.
func (*MaxExecutionTimeRule) Name() string {
	return "MaxExecutionTimeRule"
}

// HasSet returns whether the system variable is set.
func (r *MaxExecutionTimeRule) HasSet() bool {
	return r.hasSet
}

// OnEnter is called when entering a parse tree node.
func (r *MaxExecutionTimeRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeSimpleStatement:
		r.checkSimpleStatement(ctx.(*mysql.SimpleStatementContext))
	case NodeTypeSetStatement:
		r.checkSetStatement(ctx.(*mysql.SetStatementContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*MaxExecutionTimeRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *MaxExecutionTimeRule) checkSimpleStatement(ctx *mysql.SimpleStatementContext) {
	// Skip if we have already found the system variable is set or the statement is a SET statement.
	if r.hasSet || ctx.SetStatement() != nil {
		return
	}

	// The set max execution time statement should be the first statement in the SQL.
	// Otherwise, we will always set the advice (but only once).
	if len(r.adviceList) == 0 {
		r.setAdvice()
	}
}

func (r *MaxExecutionTimeRule) checkSetStatement(ctx *mysql.SetStatementContext) {
	startOptionValueList := ctx.StartOptionValueList()
	if ctx.StartOptionValueList() == nil {
		return
	}

	variable, value := "", ""
	optionValueList := startOptionValueList.StartOptionValueListFollowingOptionType()
	if optionValueList != nil {
		tmp := optionValueList.OptionValueFollowingOptionType()
		if tmp != nil {
			if tmp.InternalVariableName() != nil && tmp.SetExprOrDefault() != nil {
				variable, value = tmp.InternalVariableName().GetText(), tmp.SetExprOrDefault().GetText()
			}
		}
	}
	optionValueNoOptionType := startOptionValueList.OptionValueNoOptionType()
	if optionValueNoOptionType != nil {
		if optionValueNoOptionType.InternalVariableName() != nil && optionValueNoOptionType.SetExprOrDefault() != nil {
			variable, value = optionValueNoOptionType.InternalVariableName().GetText(), optionValueNoOptionType.SetExprOrDefault().GetText()
		}
	}
	_, err := strconv.Atoi(value)
	if strings.ToLower(variable) == r.systemVariable && err == nil {
		r.hasSet = true
	}
}

func (r *MaxExecutionTimeRule) setAdvice() {
	r.adviceList = append(r.adviceList, &storepb.Advice{
		Status:  r.level,
		Code:    code.StatementNoMaxExecutionTime.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf("The %s is not set", r.systemVariable),
	})
}
