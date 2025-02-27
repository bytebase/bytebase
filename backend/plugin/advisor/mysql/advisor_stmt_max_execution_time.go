package mysql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*MaxExecutionTimeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementMaxExecutionTime, &MaxExecutionTimeAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLStatementMaxExecutionTime, &MaxExecutionTimeAdvisor{})
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
	checker := &maxExecutionTimeChecker{
		level:          level,
		title:          string(checkCtx.Rule.Type),
		systemVariable: systemVariable,
		adviceList:     []*storepb.Advice{},
	}
	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}
	// If no system variable is set, we should set the advice.
	if !checker.hasSet {
		checker.setAdvice()
	}
	return checker.adviceList, nil
}

type maxExecutionTimeChecker struct {
	*mysql.BaseMySQLParserListener

	// The system variable name for max execution time.
	// For MySQL, it is `max_execution_time`.
	// For MariaDB, it is `max_statement_time`.
	systemVariable string
	hasSet         bool

	baseLine   int
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

func (checker *maxExecutionTimeChecker) EnterSimpleStatement(ctx *mysql.SimpleStatementContext) {
	// Skip if we have already found the system variable is set or the statement is a SET statement.
	if checker.hasSet || ctx.SetStatement() != nil {
		return
	}

	// The set max execution time statement should be the first statement in the SQL.
	// Otherwise, we will always set the advice.
	checker.setAdvice()
}

func (checker *maxExecutionTimeChecker) EnterSetStatement(ctx *mysql.SetStatementContext) {
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
	if strings.ToLower(variable) == checker.systemVariable && err == nil {
		checker.hasSet = true
	}
}

func (checker *maxExecutionTimeChecker) setAdvice() {
	checker.adviceList = []*storepb.Advice{
		{
			Status:  checker.level,
			Code:    advisor.StatementNoMaxExecutionTime.Int32(),
			Title:   checker.title,
			Content: fmt.Sprintf("The %s is not set", checker.systemVariable),
		},
	}
}
