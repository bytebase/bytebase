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
	redshiftdriver "github.com/bytebase/bytebase/backend/plugin/db/redshift"
	tidbdriver "github.com/bytebase/bytebase/backend/plugin/db/tidb"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
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

	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &config.InstanceId})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %v", config.InstanceId)
	}
	if instance == nil {
		return nil, errors.Errorf("instance %s not found", config.InstanceId)
	}
	if !common.StatementReportEngines[instance.Metadata.GetEngine()] {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Code:    common.Ok.Int32(),
				Title:   fmt.Sprintf("Statement report is not supported for %s", instance.Metadata.GetEngine()),
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

	// Check statement syntax error.
	_, syntaxAdvices := e.sheetManager.GetASTsForChecks(instance.Metadata.GetEngine(), statement)
	if len(syntaxAdvices) > 0 {
		advice := syntaxAdvices[0]
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   advice.Title,
				Content: advice.Content,
				Code:    advice.Code,
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:          advice.GetStartPosition().GetLine(),
						Column:        advice.GetStartPosition().GetColumn(),
						StartPosition: advice.StartPosition,
						EndPosition:   advice.EndPosition,
					},
				},
			},
		}, nil
	}

	planCheckRunResult := &storepb.PlanCheckRunResult_Result{
		Status: storepb.PlanCheckRunResult_Result_SUCCESS,
		Code:   common.Ok.Int32(),
		Title:  "OK",
	}
	summaryReport, err := GetSQLSummaryReport(ctx, e.store, e.sheetManager, e.dbFactory, database, statement)
	if err != nil {
		return nil, err
	}
	if summaryReport != nil {
		planCheckRunResult.Report = &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
			SqlSummaryReport: summaryReport,
		}
	}
	return []*storepb.PlanCheckRunResult_Result{planCheckRunResult}, nil
}

// GetSQLSummaryReport gets the SQL summary report for the given statement and database.
func GetSQLSummaryReport(ctx context.Context, stores *store.Store, sheetManager *sheet.Manager, dbFactory *dbfactory.DBFactory, database *store.DatabaseMessage, statement string) (*storepb.PlanCheckRunResult_Result_SqlSummaryReport, error) {
	databaseSchema, err := stores.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
	if err != nil {
		return nil, err
	}
	if databaseSchema == nil {
		return nil, errors.Errorf("database schema %s not found", database.String())
	}
	if databaseSchema.GetMetadata() == nil {
		return nil, errors.Errorf("database schema metadata %s not found", database.String())
	}
	instance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found: %s", database.InstanceID)
	}

	asts, syntaxAdvices := sheetManager.GetASTsForChecks(instance.Metadata.GetEngine(), statement)
	if len(syntaxAdvices) > 0 {
		// Return nil as it should already be checked before running this function.
		return nil, nil
	}

	var explainCalculator getAffectedRowsFromExplain
	var sqlTypes []string
	var defaultSchema string
	useDatabaseOwner, err := getUseDatabaseOwner(ctx, stores, instance, database)
	if err != nil {
		return nil, err
	}
	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
		UseDatabaseOwner: useDatabaseOwner,
	})
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)

	switch instance.Metadata.GetEngine() {
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
	case storepb.Engine_REDSHIFT:
		rd, ok := driver.(*redshiftdriver.Driver)
		if !ok {
			return nil, errors.Errorf("invalid redshift driver type")
		}
		explainCalculator = rd.CountAffectedRows

		// Use pg parser for Redshift.
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

		if instance.Metadata.GetEngine() != storepb.Engine_OCEANBASE {
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

		if instance.Metadata.GetEngine() != storepb.Engine_OCEANBASE_ORACLE {
			sqlTypes, err = plsql.GetStatementTypes(asts)
			if err != nil {
				return nil, err
			}
		}
		defaultSchema = database.DatabaseName
	case storepb.Engine_MSSQL:
		md, ok := driver.(*mssqldriver.Driver)
		if !ok {
			return nil, errors.Errorf("invalid mssql driver type")
		}
		explainCalculator = md.CountAffectedRows

		sqlTypes, err = tsql.GetStatementTypes(asts)
		if err != nil {
			slog.Error("failed to get statement types", log.BBError(err))
		}
		defaultSchema = "DBO"
	default:
		// Already checked in the Run().
		return nil, nil
	}

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)
	changeSummary, err := base.ExtractChangedResources(instance.Metadata.GetEngine(), database.DatabaseName, defaultSchema, databaseSchema, asts, renderedStatement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract changed resources")
	}
	totalAffectedRows := calculateAffectedRows(ctx, changeSummary, explainCalculator)

	return &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
		StatementTypes:   sqlTypes,
		AffectedRows:     int32(totalAffectedRows),
		ChangedResources: changeSummary.ChangedResources.Build(),
	}, nil
}

func getUseDatabaseOwner(ctx context.Context, stores *store.Store, instance *store.InstanceMessage, database *store.DatabaseMessage) (bool, error) {
	if instance.Metadata.GetEngine() != storepb.Engine_POSTGRES {
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
