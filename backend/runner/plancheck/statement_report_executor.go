package plancheck

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewStatementReportExecutor creates a statement report executor.
func NewStatementReportExecutor(store *store.Store, sheetManager *sheet.Manager, dbFactory *dbfactory.DBFactory) Executor {
	return &StatementReportExecutor{
		store:        store,
		sheetManager: sheetManager,
		dbFactory:    dbFactory,
	}
}

// StatementReportExecutor is the statement report executor.
type StatementReportExecutor struct {
	store        *store.Store
	sheetManager *sheet.Manager
	dbFactory    *dbfactory.DBFactory
}

// Run runs the statement report executor.
func (e *StatementReportExecutor) Run(ctx context.Context, config *storepb.PlanCheckRunConfig) ([]*storepb.PlanCheckRunResult_Result, error) {
	sheetUID := int(config.SheetUid)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %d", sheetUID)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found", sheetUID)
	}
	if sheet.Size > common.MaxSheetCheckSize {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_WARNING,
				Code:    common.SizeExceeded.Int32(),
				Title:   "Report for large SQL is not supported",
				Content: "",
			},
		}, nil
	}
	statement, err := e.store.GetSheetStatementByID(ctx, sheetUID)
	if err != nil {
		return nil, err
	}

	instanceUID := int(config.InstanceUid)
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance UID %v", instanceUID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found UID %v", instanceUID)
	}
	if !common.StatementReportEngines[instance.Engine] {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Code:    common.Ok.Int32(),
				Title:   fmt.Sprintf("Statement report is not supported for %s", instance.Engine),
				Content: "",
			},
		}, nil
	}

	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &config.DatabaseName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", config.DatabaseName)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %q", config.DatabaseName)
	}

	results, err := e.runReport(ctx, instance, database, statement)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Title:   "OK",
				Content: "",
				Code:    common.Ok.Int32(),
				Report:  nil,
			},
		}, nil
	}
	return results, nil
}

func (e *StatementReportExecutor) runReport(ctx context.Context, instance *store.InstanceMessage, database *store.DatabaseMessage, statement string) ([]*storepb.PlanCheckRunResult_Result, error) {
	databaseSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	if databaseSchema == nil {
		return nil, errors.Errorf("database schema not found: %d", database.UID)
	}
	if databaseSchema.GetMetadata() == nil {
		return nil, errors.Errorf("database schema metadata not found: %d", database.UID)
	}

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	asts, advices := e.sheetManager.GetASTsForChecks(instance.Engine, statement)
	if len(advices) > 0 {
		advice := advices[0]
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   advice.Title,
				Content: advice.Content,
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:          advice.GetStartPosition().GetLine(),
						Column:        advice.GetStartPosition().GetColumn(),
						Code:          advice.Code,
						Detail:        advice.Detail,
						StartPosition: advice.StartPosition,
						EndPosition:   advice.EndPosition,
					},
				},
			},
		}, nil
	}

	var sqlDB *sql.DB
	var explainCalculator getAffectedRowsFromExplain
	var sqlTypes []string
	var defaultSchema string
	switch instance.Engine {
	case storepb.Engine_POSTGRES:
		driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)
		sqlDB = driver.GetDB()
		explainCalculator = getAffectedRowsCountForPostgres

		sqlTypes, err = pg.GetStatementTypes(asts)
		if err != nil {
			return nil, err
		}
		defaultSchema = "public"
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_OCEANBASE:
		driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)
		sqlDB = driver.GetDB()
		if instance.Engine == storepb.Engine_OCEANBASE {
			explainCalculator = getAffectedRowsCountForOceanBase
		} else {
			explainCalculator = getAffectedRowsCountForMysql

			sqlTypes, err = mysqlparser.GetStatementTypes(asts)
			if err != nil {
				return nil, err
			}
		}
		// TODO(d): implement TiDB sqlTypes.
		defaultSchema = ""
	case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
		explainCalculator = getAffectedRowsCountForOracle
		defaultSchema = database.DatabaseName
	default:
		// Already checked in the Run().
		return nil, nil
	}

	changeSummary, err := base.ExtractChangedResources(instance.Engine, database.DatabaseName, defaultSchema, databaseSchema, asts, renderedStatement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract changed resources")
	}
	totalAffectedRows := calculateAffectedRows(ctx, changeSummary, sqlDB, explainCalculator)

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status: storepb.PlanCheckRunResult_Result_SUCCESS,
			Code:   common.Ok.Int32(),
			Title:  "OK",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					StatementTypes:   sqlTypes,
					AffectedRows:     int32(totalAffectedRows),
					ChangedResources: changeSummary.ChangedResources.Build(),
				},
			},
		},
	}, nil
}

type getAffectedRowsFromExplain func(context.Context, *sql.DB, string) (int64, error)

func calculateAffectedRows(ctx context.Context, changeSummary *base.ChangeSummary, sqlDB *sql.DB, explainCalculator getAffectedRowsFromExplain) int64 {
	var totalAffectedRows int64
	// Count DMLs.
	sampleCount := 0
	for _, dml := range changeSummary.SampleDMLS {
		count, err := explainCalculator(ctx, sqlDB, dml)
		if err != nil {
			slog.Error("failed to calculate affected rows", log.BBError(err))
			continue
		}
		sampleCount++
		totalAffectedRows += count
	}
	if sampleCount > 0 {
		totalAffectedRows = int64((float64(totalAffectedRows) / float64(sampleCount)) * float64(changeSummary.DMLCount))
	}
	totalAffectedRows += int64(changeSummary.InsertCount)
	// Count affected rows by DDLs.
	totalAffectedRows += changeSummary.ChangedResources.CountAffectedTableRows()

	return totalAffectedRows
}
