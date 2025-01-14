package plancheck

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mssqldriver "github.com/bytebase/bytebase/backend/plugin/db/mssql"
	mysqldriver "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	oracledriver "github.com/bytebase/bytebase/backend/plugin/db/oracle"
	pgdriver "github.com/bytebase/bytebase/backend/plugin/db/pg"
	tidbdriver "github.com/bytebase/bytebase/backend/plugin/db/tidb"
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

	var explainCalculator getAffectedRowsFromExplain
	var sqlTypes []string
	var defaultSchema string
	useDatabaseOwner, err := getUseDatabaseOwner(ctx, e.store, instance, database)
	if err != nil {
		return nil, err
	}
	driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
		UseDatabaseOwner: useDatabaseOwner,
	})
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)

	switch instance.Engine {
	case storepb.Engine_POSTGRES:
		pd, ok := driver.(*pgdriver.Driver)
		if !ok {
			return nil, errors.Errorf("invalid pg driver type")
		}
		explainCalculator = pd.CountAffectedRows

		sqlTypes, err = pg.GetStatementTypes(asts)
		if err != nil {
			return nil, err
		}
		defaultSchema = "public"
	case storepb.Engine_MYSQL, storepb.Engine_OCEANBASE:
		md, ok := driver.(*mysqldriver.Driver)
		if !ok {
			return nil, errors.Errorf("invalid mysql driver type")
		}
		explainCalculator = md.CountAffectedRows

		if instance.Engine != storepb.Engine_OCEANBASE {
			sqlTypes, err = mysqlparser.GetStatementTypes(asts)
			if err != nil {
				return nil, err
			}
		}
		defaultSchema = ""
	case storepb.Engine_TIDB:
		md, ok := driver.(*tidbdriver.Driver)
		if !ok {
			return nil, errors.Errorf("invalid tidb driver type")
		}
		explainCalculator = md.CountAffectedRows

		// TODO(d): implement TiDB sqlTypes.
		sqlTypes, err = mysqlparser.GetStatementTypes(asts)
		if err != nil {
			slog.Error("failed to get statement types", log.BBError(err))
		}
		defaultSchema = ""
	case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
		od, ok := driver.(*oracledriver.Driver)
		if !ok {
			return nil, errors.Errorf("invalid oracle driver type")
		}
		explainCalculator = od.CountAffectedRows

		defaultSchema = database.DatabaseName
	case storepb.Engine_MSSQL:
		md, ok := driver.(*mssqldriver.Driver)
		if !ok {
			return nil, errors.Errorf("invalid mssql driver type")
		}
		explainCalculator = md.CountAffectedRows

		defaultSchema = "DBO"
	default:
		// Already checked in the Run().
		return nil, nil
	}

	changeSummary, err := base.ExtractChangedResources(instance.Engine, database.DatabaseName, defaultSchema, databaseSchema, asts, renderedStatement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract changed resources")
	}
	totalAffectedRows := calculateAffectedRows(ctx, changeSummary, explainCalculator)

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

func getUseDatabaseOwner(ctx context.Context, stores *store.Store, instance *store.InstanceMessage, database *store.DatabaseMessage) (bool, error) {
	if instance.Engine != storepb.Engine_POSTGRES {
		return false, nil
	}

	// Check the project setting to see if we should use the database owner.
	project, err := stores.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get project")
	}

	if project.Setting == nil {
		return false, nil
	}

	return project.Setting.PostgresDatabaseTenantMode, nil
}

type getAffectedRowsFromExplain func(context.Context, string) (int64, error)

func calculateAffectedRows(ctx context.Context, changeSummary *base.ChangeSummary, explainCalculator getAffectedRowsFromExplain) int64 {
	var totalAffectedRows int64
	// Count DMLs.
	sampleCount := 0
	if explainCalculator != nil {
		for _, dml := range changeSummary.SampleDMLS {
			count, err := explainCalculator(ctx, dml)
			if err != nil {
				slog.Error("failed to calculate affected rows", log.BBError(err))
				continue
			}
			sampleCount++
			totalAffectedRows += count
		}
	}
	if sampleCount > 0 {
		totalAffectedRows = int64((float64(totalAffectedRows) / float64(sampleCount)) * float64(changeSummary.DMLCount))
	}
	totalAffectedRows += int64(changeSummary.InsertCount)
	// Count affected rows by DDLs.
	totalAffectedRows += changeSummary.ChangedResources.CountAffectedTableRows()

	return totalAffectedRows
}
