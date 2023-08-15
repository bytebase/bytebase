package plancheck

import (
	"context"
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
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var _ Executor = (*StatementTypeExecutor)(nil)

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
func (e *StatementTypeExecutor) Run(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) ([]*storepb.PlanCheckRunResult_Result, error) {
	changeType := planCheckRun.Config.ChangeDatabaseType
	if changeType == storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED {
		return nil, errors.Errorf("change database type is unspecified")
	}

	databaseID := int(planCheckRun.Config.DatabaseId)
	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseID})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, errors.Errorf("database not found: %d", databaseID)
	}

	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %v", database.InstanceID)
	}

	dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	if dbSchema == nil {
		return nil, errors.Errorf("database schema not found: %d", database.UID)
	}

	sheetID := int(planCheckRun.Config.SheetId)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetID}, api.SystemBotID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %d", sheetID)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found", sheetID)
	}
	if sheet.Size > common.MaxSheetSizeForTaskCheck {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Code:    common.Ok.Int64(),
				Title:   "Large SQL review policy is disabled",
				Content: "",
			},
		}, nil
	}

	statement, err := e.store.GetSheetStatementByID(ctx, sheetID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", sheetID)
	}

	materials := backendutils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := backendutils.RenderStatement(statement, materials)

	var results []*storepb.PlanCheckRunResult_Result
	switch instance.Engine {
	case db.Postgres, db.RisingWave:
		checkResults, err := postgresqlStatementTypeCheck(renderedStatement, changeType)
		if err != nil {
			return nil, err
		}
		results = append(results, checkResults...)
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		checkResults, err := mysqlStatementTypeCheck(renderedStatement, dbSchema.Metadata.CharacterSet, dbSchema.Metadata.Collation, changeType)
		if err != nil {
			return nil, err
		}
		results = append(results, checkResults...)
		if changeType == storepb.PlanCheckRunConfig_SDL {
			sdlAdvice, err := e.mysqlSDLTypeCheck(ctx, renderedStatement, instance, database)
			if err != nil {
				return nil, err
			}
			results = append(results, sdlAdvice...)
		}
	default:
		return nil, common.Errorf(common.Invalid, "invalid check statement type database type: %s", instance.Engine)
	}

	if len(results) == 0 {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Title:   "OK",
				Content: "",
				Code:    common.Ok.Int64(),
				Report:  nil,
			},
		}, nil
	}

	return results, nil
}

func (e *StatementTypeExecutor) mysqlSDLTypeCheck(ctx context.Context, newSchema string, instance *store.InstanceMessage, database *store.DatabaseMessage) ([]*storepb.PlanCheckRunResult_Result, error) {
	ddl, err := utils.ComputeDatabaseSchemaDiff(ctx, instance, database, e.dbFactory, newSchema)
	if err != nil {
		return nil, err
	}

	list, err := parser.SplitMultiSQL(parser.MySQL, ddl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split SQL")
	}

	var results []*storepb.PlanCheckRunResult_Result
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
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Code:    common.TaskTypeDropTable.Int64(),
					Title:   "Plan to drop table",
					Content: fmt.Sprintf("Plan to drop table `%s`", table.Name.O),
					Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
						SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
							Line:   int64(stmt.LastLine),
							Detail: "",
							Code:   0,
						},
					},
				})
			}
		case *tidbast.DropIndexStmt:
			results = append(results, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.PlanCheckRunResult_Result_WARNING,
				Code:    common.TaskTypeDropIndex.Int64(),
				Title:   "Plan to drop index",
				Content: fmt.Sprintf("Plan to drop index `%s` on table `%s`", node.IndexName, node.Table.Name.O),
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:   int64(stmt.LastLine),
						Detail: "",
						Code:   0,
					},
				},
			})
		case *tidbast.AlterTableStmt:
			for _, spec := range node.Specs {
				switch spec.Tp {
				case tidbast.AlterTableDropColumn:
					results = append(results, &storepb.PlanCheckRunResult_Result{
						Status:  storepb.PlanCheckRunResult_Result_WARNING,
						Code:    common.TaskTypeDropColumn.Int64(),
						Title:   "Plan to drop column",
						Content: fmt.Sprintf("Plan to drop column `%s` on table `%s`", spec.OldColumnName.Name.O, node.Table.Name.O),
						Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
							SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
								Line:   int64(stmt.LastLine),
								Detail: "",
								Code:   0,
							},
						},
					})
				case tidbast.AlterTableDropPrimaryKey:
					results = append(results, &storepb.PlanCheckRunResult_Result{
						Status:  storepb.PlanCheckRunResult_Result_WARNING,
						Code:    common.TaskTypeDropPrimaryKey.Int64(),
						Title:   "Plan to drop primary key",
						Content: fmt.Sprintf("Plan to drop primary key on table `%s`", node.Table.Name.O),
						Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
							SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
								Line:   int64(stmt.LastLine),
								Detail: "",
								Code:   0,
							},
						},
					})
				case tidbast.AlterTableDropForeignKey:
					results = append(results, &storepb.PlanCheckRunResult_Result{
						Status:  storepb.PlanCheckRunResult_Result_WARNING,
						Code:    common.TaskTypeDropForeignKey.Int64(),
						Title:   "Plan to drop foreign key",
						Content: fmt.Sprintf("Plan to drop foreign key `%s` on table `%s`", spec.Name, node.Table.Name.O),
						Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
							SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
								Line:   int64(stmt.LastLine),
								Detail: "",
								Code:   0,
							},
						},
					})
				case tidbast.AlterTableDropCheck:
					results = append(results, &storepb.PlanCheckRunResult_Result{
						Status:  storepb.PlanCheckRunResult_Result_WARNING,
						Code:    common.TaskTypeDropCheck.Int64(),
						Title:   "Plan to drop check constraint",
						Content: fmt.Sprintf("Plan to drop check constraint `%s` on table `%s`", spec.Constraint.Name, node.Table.Name.O),
						Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
							SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
								Line:   int64(stmt.LastLine),
								Detail: "",
								Code:   0,
							},
						},
					})
				}
			}
		}
	}
	return results, nil
}

func mysqlCreateAndDropDatabaseCheck(nodeList []tidbast.StmtNode) []*storepb.PlanCheckRunResult_Result {
	var results []*storepb.PlanCheckRunResult_Result
	for _, node := range nodeList {
		switch node.(type) {
		case *tidbast.DropDatabaseStmt:
			results = append(results, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    common.TaskTypeDropDatabase.Int64(),
				Title:   "Cannot drop database",
				Content: fmt.Sprintf(`The statement "%s" drops database`, node.Text()),
			})
		case *tidbast.CreateDatabaseStmt:
			results = append(results, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    common.TaskTypeCreateDatabase.Int64(),
				Title:   "Cannot create database",
				Content: fmt.Sprintf(`The statement "%s" creates database`, node.Text()),
			})
		}
	}

	return results
}

func mysqlStatementTypeCheck(statement string, charset string, collation string, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) ([]*storepb.PlanCheckRunResult_Result, error) {
	// Due to the limitation of TiDB parser, we should split the multi-statement into single statements, and extract
	// the TiDB unsupported statements, otherwise, the parser will panic or return the error.
	unsupportStmt, supportStmt, err := parser.ExtractTiDBUnsupportStmts(statement)
	if err != nil {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: err.Error(),
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:   0,
						Detail: "",
						Code:   advisor.StatementSyntaxError.Int64(),
					},
				},
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
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: err.Error(),
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:   0,
						Detail: "",
						Code:   advisor.StatementSyntaxError.Int64(),
					},
				},
			},
		}, nil
	}

	if err != nil {
		// nolint: nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: err.Error(),
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:   0,
						Detail: "",
						Code:   advisor.StatementSyntaxError.Int64(),
					},
				},
			},
		}, nil
	}

	var results []*storepb.PlanCheckRunResult_Result

	// Disallow CREATE/DROP DATABASE statements.
	results = append(results, mysqlCreateAndDropDatabaseCheck(stmts)...)

	switch changeType {
	case storepb.PlanCheckRunConfig_DML:
		for _, node := range stmts {
			_, isDDL := node.(tidbast.DDLNode)
			// We only want to disallow DDL statements in CHANGE DATA.
			// We need to run some common statements, e.g. COMMIT.
			if isDDL || hasUnsupportDDL {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Title:   "Data change can only run DML",
					Content: fmt.Sprintf("\"%s\" is not DML", node.Text()),
					Code:    common.TaskTypeNotDML.Int64(),
					Report:  nil,
				})
			}
		}
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL:
		for _, node := range stmts {
			_, isDML := node.(tidbast.DMLNode)
			_, isExplain := node.(*tidbast.ExplainStmt)
			if isDML || isExplain {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Title:   "Alter schema can only run DDL",
					Content: fmt.Sprintf("\"%s\" is not DDL", node.Text()),
					Code:    common.TaskTypeNotDDL.Int64(),
					Report:  nil,
				})
			}
		}
	default:
		return nil, common.Errorf(common.Invalid, "invalid check statement type task type: %s", changeType)
	}

	return results, nil
}

func postgresqlCreateAndDropDatabaseCheck(nodeList []ast.Node) []*storepb.PlanCheckRunResult_Result {
	var result []*storepb.PlanCheckRunResult_Result
	for _, node := range nodeList {
		switch node.(type) {
		case *ast.DropDatabaseStmt:
			result = append(result, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    common.TaskTypeDropDatabase.Int64(),
				Title:   "Cannot drop database",
				Content: fmt.Sprintf(`The statement "%s" drops database`, node.Text()),
				Report:  nil,
			})
		case *ast.CreateDatabaseStmt:
			result = append(result, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    common.TaskTypeCreateDatabase.Int64(),
				Title:   "Cannot create database",
				Content: fmt.Sprintf(`The statement "%s" creates database`, node.Text()),
				Report:  nil,
			})
		}
	}
	return result
}

func postgresqlStatementTypeCheck(statement string, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) ([]*storepb.PlanCheckRunResult_Result, error) {
	stmts, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statement)
	if err != nil {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    0,
				Title:   "Syntax error",
				Content: err.Error(),
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:   0,
						Detail: "",
						Code:   advisor.StatementSyntaxError.Int64(),
					},
				},
			},
		}, nil
	}

	var results []*storepb.PlanCheckRunResult_Result

	// Disallow CREATE/DROP DATABASE statements.
	results = append(results, postgresqlCreateAndDropDatabaseCheck(stmts)...)

	switch changeType {
	case storepb.PlanCheckRunConfig_DML:
		for _, node := range stmts {
			_, isDDL := node.(ast.DDLNode)
			// We only want to disallow DDL statements in CHANGE DATA.
			// We need to run some common statements, e.g. COMMIT.
			if isDDL {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Title:   "Data change can only run DML",
					Content: fmt.Sprintf("\"%s\" is not DML", node.Text()),
					Code:    common.TaskTypeNotDML.Int64(),
					Report:  nil,
				})
			}
		}
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL:
		for _, node := range stmts {
			_, isDML := node.(ast.DMLNode)
			_, isSelect := node.(*ast.SelectStmt)
			_, isExplain := node.(*ast.ExplainStmt)
			if isDML || isSelect || isExplain {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Title:   "Alter schema can only run DDL",
					Content: fmt.Sprintf("\"%s\" is not DDL", node.Text()),
					Code:    common.TaskTypeNotDDL.Int64(),
					Report:  nil,
				})
			}
		}
	default:
		return nil, common.Errorf(common.Invalid, "invalid check statement type task type: %s", changeType)
	}

	return results, nil
}
