package taskcheck

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
	"github.com/bytebase/bytebase/store"
)

// NewStatementTypeExecutor creates a task check DML executor.
func NewStatementTypeExecutor(store *store.Store) Executor {
	return &StatementTypeExecutor{
		store: store,
	}
}

// StatementTypeExecutor is the task check DML executor.
type StatementTypeExecutor struct {
	store *store.Store
}

// Run will run the task check database connector executor once.
func (e *StatementTypeExecutor) Run(ctx context.Context, taskCheckRun *api.TaskCheckRun, task *api.Task) (result []api.TaskCheckResult, err error) {
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
	// Due to the limitation of TiDB parser, we should split the multi-statement into single statements, and extract
	// the TiDB unsupported statements, otherwise, the parser will panic or return the error.
	unsupportStmt, supportStmt, err := parser.ExtractTiDBUnsupportStmts(statement)
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
	// TODO(zp): We regard the DELIMITER statement as a DDL statement here.
	// But we should ban the DELIMITER statement because go-sql-driver doesn't support it.
	hasUnsupportDDL := len(unsupportStmt) > 0

	p := tidbparser.New()
	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	stmts, _, err := p.Parse(supportStmt, charset, collation)
	if err != nil {
		//nolint: nilerr
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

	if err != nil {
		//nolint: nilerr
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
			// We only want to disallow DDL statements in CHANGE DATA.
			// We need to run some common statements, e.g. COMMIT.
			if isDDL || hasUnsupportDDL {
				result = append(result, api.TaskCheckResult{
					Status:    api.TaskCheckStatusWarn,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDML.Int(),
					Title:     "Data change can only run DML",
					Content:   fmt.Sprintf("\"%s\" is not DML", node.Text()),
				})
			}
		}
	case api.TaskDatabaseSchemaUpdate, api.TaskDatabaseSchemaUpdateSDL, api.TaskDatabaseSchemaUpdateGhostSync:
		for _, node := range stmts {
			_, isDML := node.(tidbast.DMLNode)
			_, isExplain := node.(*tidbast.ExplainStmt)
			if isDML || isExplain {
				result = append(result, api.TaskCheckResult{
					Status:    api.TaskCheckStatusWarn,
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
	stmts, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statement)
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
			// We only want to disallow DDL statements in CHANGE DATA.
			// We need to run some common statements, e.g. COMMIT.
			if isDDL {
				result = append(result, api.TaskCheckResult{
					Status:    api.TaskCheckStatusWarn,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeNotDML.Int(),
					Title:     "Data change can only run DML",
					Content:   fmt.Sprintf("\"%s\" is not DML", node.Text()),
				})
			}
		}
	case api.TaskDatabaseSchemaUpdate, api.TaskDatabaseSchemaUpdateSDL, api.TaskDatabaseSchemaUpdateGhostSync:
		for _, node := range stmts {
			_, isDML := node.(ast.DMLNode)
			_, isSelect := node.(*ast.SelectStmt)
			_, isExplain := node.(*ast.ExplainStmt)
			if isDML || isSelect || isExplain {
				result = append(result, api.TaskCheckResult{
					Status:    api.TaskCheckStatusWarn,
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
