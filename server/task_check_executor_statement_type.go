package server

import (
	"context"
	"encoding/json"
	"fmt"

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
		return []api.TaskCheckResult{}, common.WithError(common.Internal, err)
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
		return nil, common.Errorf(common.Invalid, "invalid check statement type payload: %w", err)
	}

	if payload.DbType != db.Postgres {
		return nil, common.Errorf(common.Invalid, "invalid check statement type database type: %s", payload.DbType)
	}

	stmts, err := parser.Parse(parser.Postgres, parser.Context{}, payload.Statement)
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

	switch task.Type {
	case api.TaskDatabaseDataUpdate:
		for _, node := range stmts {
			if _, ok := node.(ast.DMLNode); !ok {
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
		return nil, common.Errorf(common.Invalid, "invalid check statement type task type: %s", task.Type)
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
