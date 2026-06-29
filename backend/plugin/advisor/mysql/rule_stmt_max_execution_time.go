package mysql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*MaxExecutionTimeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME, &MaxExecutionTimeAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME, &MaxExecutionTimeAdvisor{})
}

// MaxExecutionTimeAdvisor is the advisor checking for the max execution time.
type MaxExecutionTimeAdvisor struct {
}

func (*MaxExecutionTimeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	systemVariable := "max_execution_time"
	if checkCtx.Rule.Engine == storepb.Engine_MARIADB {
		systemVariable = "max_statement_time"
	}

	hasSet := false
	var allAdvice []*storepb.Advice

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}

		switch n := node.(type) {
		case *ast.SetStmt:
			for _, assign := range n.Assignments {
				if assign.Column == nil || assign.Value == nil {
					continue
				}
				varName := assign.Column.Column
				if strings.ToLower(varName) == systemVariable {
					if intLit, ok := assign.Value.(*ast.IntLit); ok {
						_ = intLit
						hasSet = true
					} else if str, ok := assign.Value.(*ast.StringLit); ok {
						if _, err := strconv.Atoi(str.Value); err == nil {
							hasSet = true
						}
					}
				}
			}
		default:
			if !hasSet && len(allAdvice) == 0 {
				allAdvice = append(allAdvice, &storepb.Advice{
					Status:  level,
					Code:    code.StatementNoMaxExecutionTime.Int32(),
					Title:   checkCtx.Rule.Type.String(),
					Content: fmt.Sprintf("The %s is not set", systemVariable),
				})
			}
		}
	}

	return allAdvice, nil
}
