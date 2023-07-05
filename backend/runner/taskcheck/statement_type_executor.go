package taskcheck

import (
	"context"
	"encoding/json"
	"fmt"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/runner/utils"
	"github.com/bytebase/bytebase/backend/store"
	backendutils "github.com/bytebase/bytebase/backend/utils"
)

// NewStatementTypeExecutor creates a task check DML executor.
func NewStatementTypeExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) Executor {
	return &StatementTypeExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// StatementTypeExecutor is the task check DML executor.
type StatementTypeExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run will run the task check database connector executor once.
func (exec *StatementTypeExecutor) Run(ctx context.Context, _ *store.TaskCheckRunMessage, task *store.TaskMessage) (result []api.TaskCheckResult, err error) {
	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	dbSchema, err := exec.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	payload := &TaskPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, err
	}
	sheet, err := exec.store.GetSheet(ctx, &store.FindSheetMessage{UID: &payload.SheetID}, api.SystemBotID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %d", payload.SheetID)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found", payload.SheetID)
	}
	if sheet.Size > common.MaxSheetSizeForTaskCheck {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusSuccess,
				Namespace: api.AdvisorNamespace,
				Code:      common.Ok.Int(),
				Title:     "Large SQL review policy is disabled",
				Content:   "",
			},
		}, nil
	}
	statement, err := exec.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", payload.SheetID)
	}

	materials := backendutils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := backendutils.RenderStatement(statement, materials)

	switch instance.Engine {
	case db.Postgres:
		result, err = postgresqlStatementTypeCheck(renderedStatement, task.Type)
		if err != nil {
			return nil, err
		}
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		result, err = mysqlStatementTypeCheck(renderedStatement, dbSchema.Metadata.CharacterSet, dbSchema.Metadata.Collation, task.Type)
		if err != nil {
			return nil, err
		}
		if task.Type == api.TaskDatabaseSchemaUpdateSDL {
			sdlAdvice, err := exec.mysqlSDLTypeCheck(ctx, renderedStatement, task)
			if err != nil {
				return nil, err
			}
			result = append(result, sdlAdvice...)
		}
	default:
		return nil, common.Errorf(common.Invalid, "invalid check statement type database type: %s", instance.Engine)
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

func (exec *StatementTypeExecutor) mysqlSDLTypeCheck(ctx context.Context, newSchema string, task *store.TaskMessage) ([]api.TaskCheckResult, error) {
	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	ddl, err := utils.ComputeDatabaseSchemaDiff(ctx, instance, database, exec.dbFactory, newSchema)
	if err != nil {
		return nil, err
	}

	list, err := parser.SplitMultiSQL(parser.MySQL, ddl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split SQL")
	}

	var result []api.TaskCheckResult
	for _, stmt := range list {
		if parser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			continue
		}
		nodeList, _, err := tidbparser.New().Parse(stmt.Text, "", "")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse schema %q", stmt.Text)
		}
		if len(nodeList) != 1 {
			return nil, errors.Errorf("Expect one statement after splitting but found %d", len(nodeList))
		}

		switch node := nodeList[0].(type) {
		case *tidbast.DropTableStmt:
			for _, table := range node.Tables {
				result = append(result, api.TaskCheckResult{
					Status:    api.TaskCheckStatusWarn,
					Namespace: api.BBNamespace,
					Code:      common.TaskTypeDropTable.Int(),
					Title:     "Plan to drop table",
					Content:   fmt.Sprintf("Plan to drop table `%s`", table.Name.O),
					Line:      stmt.LastLine,
				})
			}
		case *tidbast.DropIndexStmt:
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusWarn,
				Namespace: api.BBNamespace,
				Code:      common.TaskTypeDropIndex.Int(),
				Title:     "Plan to drop index",
				Content:   fmt.Sprintf("Plan to drop index `%s` on table `%s`", node.IndexName, node.Table.Name.O),
				Line:      stmt.LastLine,
			})
		case *tidbast.AlterTableStmt:
			for _, spec := range node.Specs {
				switch spec.Tp {
				case tidbast.AlterTableDropColumn:
					result = append(result, api.TaskCheckResult{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.BBNamespace,
						Code:      common.TaskTypeDropColumn.Int(),
						Title:     "Plan to drop column",
						Content:   fmt.Sprintf("Plan to drop column `%s` on table `%s`", spec.OldColumnName.Name.O, node.Table.Name.O),
						Line:      stmt.LastLine,
					})
				case tidbast.AlterTableDropPrimaryKey:
					result = append(result, api.TaskCheckResult{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.BBNamespace,
						Code:      common.TaskTypeDropPrimaryKey.Int(),
						Title:     "Plan to drop primary key",
						Content:   fmt.Sprintf("Plan to drop primary key on table `%s`", node.Table.Name.O),
						Line:      stmt.LastLine,
					})
				case tidbast.AlterTableDropForeignKey:
					result = append(result, api.TaskCheckResult{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.BBNamespace,
						Code:      common.TaskTypeDropPrimaryKey.Int(),
						Title:     "Plan to drop foreign key",
						Content:   fmt.Sprintf("Plan to drop foreign key `%s` on table `%s`", spec.Name, node.Table.Name.O),
						Line:      stmt.LastLine,
					})
				case tidbast.AlterTableDropCheck:
					result = append(result, api.TaskCheckResult{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.BBNamespace,
						Code:      common.TaskTypeDropPrimaryKey.Int(),
						Title:     "Plan to drop check constraint",
						Content:   fmt.Sprintf("Plan to drop check constraint `%s` on table `%s`", spec.Constraint.Name, node.Table.Name.O),
						Line:      stmt.LastLine,
					})
				}
			}
		}
	}
	return result, nil
}

func mysqlCreateAndDropDatabaseCheck(nodeList []tidbast.StmtNode) []api.TaskCheckResult {
	var result []api.TaskCheckResult
	for _, node := range nodeList {
		switch node.(type) {
		case *tidbast.DropDatabaseStmt:
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.TaskTypeDropDatabase.Int(),
				Title:     "Cannot drop database",
				Content:   fmt.Sprintf(`The statement "%s" drops database`, node.Text()),
			})
		case *tidbast.CreateDatabaseStmt:
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.TaskTypeCreateDatabase.Int(),
				Title:     "Cannot create database",
				Content:   fmt.Sprintf(`The statement "%s" creates database`, node.Text()),
			})
		}
	}

	return result
}

func mysqlStatementTypeCheck(statement string, charset string, collation string, taskType api.TaskType) (result []api.TaskCheckResult, err error) {
	// Due to the limitation of TiDB parser, we should split the multi-statement into single statements, and extract
	// the TiDB unsupported statements, otherwise, the parser will panic or return the error.
	unsupportStmt, supportStmt, err := parser.ExtractTiDBUnsupportStmts(statement)
	if err != nil {
		// nolint:nilerr
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
		// nolint: nilerr
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
		// nolint: nilerr
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

	// Disallow CREATE/DROP DATABASE statements.
	result = append(result, mysqlCreateAndDropDatabaseCheck(stmts)...)

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

func postgresqlCreateAndDropDatabaseCheck(nodeList []ast.Node) []api.TaskCheckResult {
	var result []api.TaskCheckResult
	for _, node := range nodeList {
		switch node.(type) {
		case *ast.DropDatabaseStmt:
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.TaskTypeDropDatabase.Int(),
				Title:     "Cannot drop database",
				Content:   fmt.Sprintf(`The statement "%s" drops database`, node.Text()),
			})
		case *ast.CreateDatabaseStmt:
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.TaskTypeCreateDatabase.Int(),
				Title:     "Cannot create database",
				Content:   fmt.Sprintf(`The statement "%s" creates database`, node.Text()),
			})
		}
	}
	return result
}

func postgresqlStatementTypeCheck(statement string, taskType api.TaskType) (result []api.TaskCheckResult, err error) {
	stmts, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statement)
	if err != nil {
		// nolint:nilerr
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

	// Disallow CREATE/DROP DATABASE statements.
	result = append(result, postgresqlCreateAndDropDatabaseCheck(stmts)...)

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
