package server

import (
	"context"
	"encoding/json"
	"fmt"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

// NewTaskCheckStatementTypeExecutor creates a task check DML executor.
func NewTaskCheckStatementTypeExecutor() TaskCheckExecutor {
	return &TaskCheckStatementTypeExecutor{}
}

// TaskCheckStatementTypeExecutor is the task check DML executor.
type TaskCheckStatementTypeExecutor struct {
}

// Run will run the task check database connector executor once.
func (*TaskCheckStatementTypeExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	task, err := server.store.GetTaskByID(ctx, taskCheckRun.TaskID)
	if err != nil {
		return []api.TaskCheckResult{}, common.Wrap(err, common.Internal)
	}
	if task == nil {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.Internal.Int(),
				Title:     fmt.Sprintf("Failed to find task %v", taskCheckRun.TaskID),
				Content:   err.Error(),
			},
		}, nil
	}

	payload := &api.TaskCheckDatabaseStatementTypePayload{}
	if err := json.Unmarshal([]byte(taskCheckRun.Payload), payload); err != nil {
		return nil, common.Wrapf(err, common.Invalid, "invalid check statement type payload")
	}

	switch payload.DbType {
	case db.Postgres:
		result, err = postgresqlStatementTypeCheck(payload.Statement, task.Type)
		if err != nil {
			return nil, err
		}
	case db.MySQL, db.TiDB:
		result, err = mysqlStatementTypeCheck(payload.Statement, payload.Charset, payload.Collation, task.Type)
		if err != nil {
			return nil, err
		}
	default:
		return nil, common.Errorf(common.Invalid, "invalid check statement type database type: %s", payload.DbType)
	}

	if len(result) == 0 {
		result = append(result, api.TaskCheckResult{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   "",
		})
	}

	return result, nil
}

func mysqlStatementTypeCheck(statement string, charset string, collation string, taskType api.TaskType) (result []api.TaskCheckResult, err error) {
	p := tidbparser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	stmts, _, err := p.Parse(statement, charset, collation)
	if err != nil {
		//nolint:nilerr
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.AdvisorNamespace,
				Code:      advisor.StatementSyntaxError.Int(),
				Title:     "Syntax error",
				Content:   err.Error(),
			},
		}, nil
	}

	switch taskType {
	case api.TaskDatabaseDataUpdate:
		for _, node := range stmts {
			_, isDDL := node.(tidbast.DDLNode)
			_, isQuery := node.(*tidbast.SelectStmt)
			_, isExplain := node.(*tidbast.ExplainStmt)
			// We only want to disallow DDL, QUERY and EXPLAIN statements in CHANGE DATA.
			// We need to run some common statements, e.g. COMMIT.
			if isDDL || isQuery || isExplain {
				result = append(result, api.TaskCheckResult{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDML.Int(),
					Title:     "Data change can only run DML",
					Content:   fmt.Sprintf("\"%s\" is not DML", node.Text()),
				})
			}
		}
	case api.TaskDatabaseSchemaUpdate, api.TaskDatabaseSchemaUpdateGhostSync:
		for _, node := range stmts {
			_, isDML := node.(tidbast.DMLNode)
			_, isExplain := node.(*tidbast.ExplainStmt)
			if isDML || isExplain {
				result = append(result, api.TaskCheckResult{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDDL.Int(),
					Title:     "Alter schema can only run DDL",
					Content:   fmt.Sprintf("\"%s\" is not DDL", node.Text()),
				})
			}
		}
	default:
		return nil, common.Errorf(common.Invalid, "invalid check statement type task type: %s", taskType)
	}

	return result, nil
}

func postgresqlStatementTypeCheck(statement string, taskType api.TaskType) (result []api.TaskCheckResult, err error) {
	stmts, err := parser.Parse(parser.Postgres, parser.Context{}, statement)
	if err != nil {
		//nolint:nilerr
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.AdvisorNamespace,
				Code:      advisor.StatementSyntaxError.Int(),
				Title:     "Syntax error",
				Content:   err.Error(),
			},
		}, nil
	}

	switch taskType {
	case api.TaskDatabaseDataUpdate:
		for _, node := range stmts {
			_, isDDL := node.(ast.DDLNode)
			_, isSelect := node.(*ast.SelectStmt)
			_, isExplain := node.(*ast.ExplainStmt)
			// We only want to disallow DDL, QUERY and EXPLAIN statements in CHANGE DATA.
			// We need to run some common statements, e.g. COMMIT.
			if isDDL || isSelect || isExplain {
				result = append(result, api.TaskCheckResult{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDML.Int(),
					Title:     "Data change can only run DML",
					Content:   fmt.Sprintf("\"%s\" is not DML", node.Text()),
				})
			}
		}
	case api.TaskDatabaseSchemaUpdate, api.TaskDatabaseSchemaUpdateGhostSync:
		for _, node := range stmts {
			_, isDML := node.(ast.DMLNode)
			_, isSelect := node.(*ast.SelectStmt)
			_, isExplain := node.(*ast.ExplainStmt)
			if isDML || isSelect || isExplain {
				result = append(result, api.TaskCheckResult{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDDL.Int(),
					Title:     "Alter schema can only run DDL",
					Content:   fmt.Sprintf("\"%s\" is not DDL", node.Text()),
				})
			}
		}
	default:
		return nil, common.Errorf(common.Invalid, "invalid check statement type task type: %s", taskType)
	}

	return result, nil
}
