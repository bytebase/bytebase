package taskcheck

import (
	"context"
	"encoding/json"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewStatementTypeReportExecutor creates a task check statement type report executor.
func NewStatementTypeReportExecutor(store *store.Store) Executor {
	return &StatementTypeReportExecutor{
		store: store,
	}
}

// StatementTypeReportExecutor is the task check statement type report executor. It reports the type of each statement.
type StatementTypeReportExecutor struct {
	store *store.Store
}

// Run will run the task check statement type report executor once.
func (s *StatementTypeReportExecutor) Run(ctx context.Context, _ *store.TaskCheckRunMessage, task *store.TaskMessage) ([]api.TaskCheckResult, error) {
	if !api.IsTaskCheckReportNeededForTaskType(task.Type) {
		return nil, nil
	}
	payload := &TaskPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, err
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	if !api.IsTaskCheckReportSupported(instance.Engine) {
		return nil, nil
	}
	sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{UID: &payload.SheetID}, api.SystemBotID)
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
	statement, err := s.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", payload.SheetID)
	}
	var charset, collation string
	if task.DatabaseID != nil {
		dbSchema, err := s.store.GetDBSchema(ctx, *task.DatabaseID)
		if err != nil {
			return nil, err
		}
		charset = dbSchema.Metadata.CharacterSet
		collation = dbSchema.Metadata.Collation
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	switch instance.Engine {
	case db.Postgres:
		return reportStatementTypeForPostgres(renderedStatement)
	case db.MySQL, db.MariaDB, db.OceanBase, db.TiDB:
		return reportStatementTypeForMySQL(renderedStatement, charset, collation)
	default:
		return nil, errors.New("unsupported db type")
	}
}

func reportStatementTypeForMySQL(statement, charset, collation string) ([]api.TaskCheckResult, error) {
	singleSQLs, err := parser.SplitMultiSQL(parser.MySQL, statement)
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

	var result []api.TaskCheckResult

	p := tidbparser.New()
	p.EnableWindowFunc(true)

	for _, stmt := range singleSQLs {
		if stmt.Empty {
			continue
		}
		if parser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusSuccess,
				Namespace: api.BBNamespace,
				Code:      common.Ok.Int(),
				Title:     "OK",
				Content:   "UNKNOWN",
			})
			continue
		}
		root, _, err := p.Parse(stmt.Text, charset, collation)
		if err != nil {
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusError,
				Namespace: api.AdvisorNamespace,
				Code:      advisor.StatementSyntaxError.Int(),
				Title:     "Syntax error",
				Content:   err.Error(),
			})
			continue
		}
		if len(root) != 1 {
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.Internal.Int(),
				Title:     "Failed to report statement type",
				Content:   "Expect to get one node from parser",
			})
			continue
		}
		sqlType := getStatementTypeFromTidbAstNode(root[0])
		result = append(result, api.TaskCheckResult{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   sqlType,
		})
	}

	return result, nil
}

func reportStatementTypeForPostgres(statement string) ([]api.TaskCheckResult, error) {
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

	var result []api.TaskCheckResult

	for _, stmt := range stmts {
		sqlType := getStatementTypeFromAstNode(stmt)
		result = append(result, api.TaskCheckResult{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   sqlType,
		})
	}

	return result, nil
}

func getStatementTypeFromTidbAstNode(node tidbast.StmtNode) string {
	switch node.(type) {
	// DDL

	// CREATE
	case *tidbast.CreateDatabaseStmt:
		return "CREATE_DATABASE"
	case *tidbast.CreateIndexStmt:
		return "CREATE_INDEX"
	case *tidbast.CreateTableStmt:
		return "CREATE_TABLE"
	case *tidbast.CreateViewStmt:
		return "CREATE_VIEW"
	case *tidbast.CreateSequenceStmt:
		return "CREATE_SEQUENCE"
	case *tidbast.CreatePlacementPolicyStmt:
		return "CREATE_PLACEMENT_POLICY"

	// DROP
	case *tidbast.DropIndexStmt:
		return "DROP_INDEX"
	case *tidbast.DropTableStmt:
		return "DROP_TABLE"
	case *tidbast.DropSequenceStmt:
		return "DROP_SEQUENCE"
	case *tidbast.DropPlacementPolicyStmt:
		return "DROP_PLACEMENT_POLICY"
	case *tidbast.DropDatabaseStmt:
		return "DROP_DATABASE"

	// ALTER
	case *tidbast.AlterTableStmt:
		return "ALTER_TABLE"
	case *tidbast.AlterSequenceStmt:
		return "ALTER_SEQUENCE"
	case *tidbast.AlterPlacementPolicyStmt:
		return "ALTER_PLACEMENT_POLICY"

	// TRUNCATE
	case *tidbast.TruncateTableStmt:
		return "TRUNCATE"

	// RENAME
	case *tidbast.RenameTableStmt:
		return "RENAME_TABLE"

	// DML

	case *tidbast.InsertStmt:
		return "INSERT"
	case *tidbast.DeleteStmt:
		return "DELETE"
	case *tidbast.UpdateStmt:
		return "UPDATE"
	}
	return "UNKNOWN"
}

func getStatementTypeFromAstNode(node ast.Node) string {
	switch node := node.(type) {
	// DDL

	// CREATE
	case *ast.CreateIndexStmt:
		return "CREATE_INDEX"
	case *ast.CreateTableStmt:
		switch node.Name.Type {
		case ast.TableTypeView:
			return "CREATE_VIEW"
		case ast.TableTypeBaseTable:
			return "CREATE_TABLE"
		}
	case *ast.CreateSequenceStmt:
		return "CREATE_SEQUENCE"
	case *ast.CreateDatabaseStmt:
		return "CREATE_DATABASE"
	case *ast.CreateSchemaStmt:
		return "CREATE_SCHEMA"
	case *ast.CreateFunctionStmt:
		return "CREATE_FUNCTION"
	case *ast.CreateTriggerStmt:
		return "CREATE_TRIGGER"
	case *ast.CreateTypeStmt:
		return "CREATE_TYPE"
	case *ast.CreateExtensionStmt:
		return "CREATE_EXTENSION"

	// DROP
	case *ast.DropColumnStmt:
		return "DROP_COLUMN"
	case *ast.DropConstraintStmt:
		return "DROP_CONSTRAINT"
	case *ast.DropDatabaseStmt:
		return "DROP_DATABASE"
	case *ast.DropDefaultStmt:
		return "DROP_DEFAULT"
	case *ast.DropExtensionStmt:
		return "DROP_EXTENSION"
	case *ast.DropFunctionStmt:
		return "DROP_FUNCTION"
	case *ast.DropIndexStmt:
		return "DROP_INDEX"
	case *ast.DropNotNullStmt:
		return "DROP_NOT_NULL"
	case *ast.DropSchemaStmt:
		return "DROP_SCHEMA"
	case *ast.DropSequenceStmt:
		return "DROP_SEQUENCE"
	case *ast.DropTableStmt:
		return "DROP_TABLE"

	case *ast.DropTriggerStmt:
		return "DROP_TRIGGER"
	case *ast.DropTypeStmt:
		return "DROP_TYPE"

	// ALTER
	case *ast.AlterColumnTypeStmt:
		return "ALTER_COLUMN_TYPE"
	case *ast.AlterSequenceStmt:
		return "ALTER_SEQUENCE"
	case *ast.AlterTableStmt:
		switch node.Table.Type {
		case ast.TableTypeView:
			return "ALTER_VIEW"
		case ast.TableTypeBaseTable:
			return "ALTER_TABLE"
		}
	case *ast.AlterTypeStmt:
		return "ALTER_TYPE"

	case *ast.AddColumnListStmt:
		return "ALTER_TABLE_ADD_COLUMN_LIST"
	case *ast.AddConstraintStmt:
		return "ALTER_TABLE_ADD_CONSTRAINT"

	// RENAME
	case *ast.RenameColumnStmt:
		return "RENAME_COLUMN"
	case *ast.RenameConstraintStmt:
		return "RENAME_CONSTRAINT"
	case *ast.RenameIndexStmt:
		return "RENAME_INDEX"
	case *ast.RenameSchemaStmt:
		return "RENAME_SCHEMA"
	case *ast.RenameTableStmt:
		switch node.Table.Type {
		case ast.TableTypeView:
			return "RENAME_VIEW"
		case ast.TableTypeBaseTable:
			return "RENAME_TABLE"
		}

	// DML

	case *ast.InsertStmt:
		return "INSERT"
	case *ast.UpdateStmt:
		return "UPDATE"
	case *ast.DeleteStmt:
		return "DELETE"
	}

	return "UNKNOWN"
}
