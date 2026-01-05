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
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	redshiftparser "github.com/bytebase/bytebase/backend/plugin/parser/redshift"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	"github.com/bytebase/bytebase/backend/store"
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

// RunForTarget runs the statement report check for a single target.
func (e *StatementReportExecutor) RunForTarget(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	fullSheet, err := e.store.GetSheetFull(ctx, target.SheetSha256)
	if err != nil {
		return nil, err
	}
	if fullSheet == nil {
		return nil, errors.Errorf("sheet full %s not found", target.SheetSha256)
	}
	if fullSheet.Size > common.MaxSheetCheckSize {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_WARNING,
				Code:    common.SizeExceeded.Int32(),
				Title:   "Report for large SQL is not supported",
				Content: "",
			},
		}, nil
	}

	instanceID, databaseName, err := common.GetInstanceDatabaseID(target.Target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse target %s", target.Target)
	}

	instance, err := e.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %v", instanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance %s not found", instanceID)
	}
	if !common.EngineSupportStatementReport(instance.Metadata.GetEngine()) {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_SUCCESS,
				Code:    common.Ok.Int32(),
				Title:   fmt.Sprintf("Statement report is not supported for %s", instance.Metadata.GetEngine()),
				Content: "",
			},
		}, nil
	}

	database, err := e.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %q", databaseName)
	}

	// Check statement syntax error.
	_, syntaxAdvices := e.sheetManager.GetStatementsForChecks(instance.Metadata.GetEngine(), fullSheet.Statement)
	if len(syntaxAdvices) > 0 {
		advice := syntaxAdvices[0]
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_ERROR,
				Title:   advice.Title,
				Content: advice.Content,
				Code:    advice.Code,
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						StartPosition: advice.StartPosition,
						EndPosition:   advice.EndPosition,
					},
				},
			},
		}, nil
	}

	planCheckRunResult := &storepb.PlanCheckRunResult_Result{
		Status: storepb.Advice_SUCCESS,
		Code:   common.Ok.Int32(),
		Title:  "OK",
	}
	summaryReport, err := GetSQLSummaryReport(ctx, e.store, e.sheetManager, e.dbFactory, database, fullSheet.Statement)
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
	databaseSchema, err := stores.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, err
	}
	if databaseSchema == nil {
		return nil, errors.Errorf("database schema %s not found", database.String())
	}
	if databaseSchema.GetProto() == nil {
		return nil, errors.Errorf("database schema metadata %s not found", database.String())
	}
	instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found: %s", database.InstanceID)
	}

	stmts, syntaxAdvices := sheetManager.GetStatementsForChecks(instance.Metadata.GetEngine(), statement)
	if len(syntaxAdvices) > 0 {
		// Return nil as it should already be checked before running this function.
		return nil, nil
	}
	asts := parserbase.ExtractASTs(stmts)

	var explainCalculator getAffectedRowsFromExplain
	var sqlTypes []string
	var defaultSchema string
	project, err := stores.GetProject(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return nil, err
	}
	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
		TenantMode: project.Setting.GetPostgresDatabaseTenantMode(),
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

		stmtsWithPos, err := pg.GetStatementTypes(asts)
		if err != nil {
			return nil, err
		}
		sqlTypes = make([]string, len(stmtsWithPos))
		for i, stmt := range stmtsWithPos {
			sqlTypes[i] = stmt.Type
		}
		defaultSchema = "public"
	case storepb.Engine_REDSHIFT:
		rd, ok := driver.(*redshiftdriver.Driver)
		if !ok {
			return nil, errors.Errorf("invalid redshift driver type")
		}
		explainCalculator = rd.CountAffectedRows

		// Use Redshift parser for statement types
		sqlTypes, err = redshiftparser.GetStatementTypes(asts)
		if err != nil {
			return nil, err
		}
		defaultSchema = "public"
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
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

		sqlTypes, err = tidbparser.GetStatementTypes(asts)
		if err != nil {
			slog.Error("failed to get statement types", log.BBError(err))
		}
		defaultSchema = ""
	case storepb.Engine_ORACLE:
		od, ok := driver.(*oracledriver.Driver)
		if !ok {
			return nil, errors.Errorf("invalid oracle driver type")
		}
		explainCalculator = od.CountAffectedRows

		sqlTypes, err = plsql.GetStatementTypes(asts)
		if err != nil {
			return nil, err
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
		defaultSchema = "dbo"
	default:
		// Already checked in the Run().
		return nil, nil
	}

	// Database secrets feature has been removed
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	// Database secrets feature removed - using original statement directly
	changeSummary, err := parserbase.ExtractChangedResources(instance.Metadata.GetEngine(), database.DatabaseName, defaultSchema, databaseSchema, asts, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract changed resources")
	}
	totalAffectedRows := calculateAffectedRows(ctx, changeSummary, explainCalculator)

	return &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
		StatementTypes:   sqlTypes,
		AffectedRows:     totalAffectedRows,
		ChangedResources: changeSummary.ChangedResources.Build(),
	}, nil
}

type getAffectedRowsFromExplain func(context.Context, string) (int64, error)

func calculateAffectedRows(ctx context.Context, changeSummary *parserbase.ChangeSummary, explainCalculator getAffectedRowsFromExplain) int64 {
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
