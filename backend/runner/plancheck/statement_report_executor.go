package plancheck

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
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
	dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	if dbSchema == nil {
		return nil, errors.Errorf("database schema not found: %d", database.UID)
	}
	if dbSchema.GetMetadata() == nil {
		return nil, errors.Errorf("database schema metadata not found: %d", database.UID)
	}

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	asts, advices := e.sheetManager.GetASTsForChecks(instance.Engine, statement)
	if len(advices) > 0 {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: advices[0].Content,
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
					SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
						Code: advisor.StatementSyntaxError.Int32(),
					},
				},
			},
		}, nil
	}

	switch instance.Engine {
	case storepb.Engine_POSTGRES:
		driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)
		sqlDB := driver.GetDB()

		return reportForPostgres(ctx, sqlDB, database.DatabaseName, asts, renderedStatement, dbSchema)
	case storepb.Engine_MYSQL, storepb.Engine_OCEANBASE:
		driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)
		sqlDB := driver.GetDB()

		return reportForMySQL(ctx, sqlDB, instance.Engine, database.DatabaseName, asts, renderedStatement, dbSchema)
	case storepb.Engine_ORACLE, storepb.Engine_DM, storepb.Engine_OCEANBASE_ORACLE:
		return reportForOracle(database.DatabaseName, database.DatabaseName, asts, renderedStatement, dbSchema)
	default:
		// Already checked in the Run().
		return nil, nil
	}
}

func reportForPostgres(ctx context.Context, sqlDB *sql.DB, database string, asts any, statement string, dbMetadata *model.DBSchema) ([]*storepb.PlanCheckRunResult_Result, error) {
	sqlTypes, err := pg.GetStatementTypes(asts)
	if err != nil {
		return nil, err
	}

	changeSummary, err := base.ExtractChangedResources(storepb.Engine_POSTGRES, database, "public", asts, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract changed resources")
	}

	totalAffectedRows, err := calculateAffectedRows(ctx, changeSummary, dbMetadata, sqlDB, getAffectedRowsCountForPostgres)
	if err != nil {
		return nil, err
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status: storepb.PlanCheckRunResult_Result_SUCCESS,
			Code:   common.Ok.Int32(),
			Title:  "OK",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					StatementTypes:   sqlTypes,
					AffectedRows:     int32(totalAffectedRows),
					ChangedResources: convertToChangedResources(dbMetadata, changeSummary.ResourceChanges),
				},
			},
		},
	}, nil
}

func reportForMySQL(ctx context.Context, sqlDB *sql.DB, engine storepb.Engine, databaseName string, asts any, statement string, dbMetadata *model.DBSchema) ([]*storepb.PlanCheckRunResult_Result, error) {
	sqlTypes, err := pg.GetStatementTypes(asts)
	if err != nil {
		return nil, err
	}

	changeSummary, err := base.ExtractChangedResources(storepb.Engine_MYSQL, databaseName, "" /* currentSchema */, asts, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract changed resources")
	}

	var explainCalculator getAffectedRowsFromExplain
	switch engine {
	case storepb.Engine_OCEANBASE:
		explainCalculator = getAffectedRowsCountForOceanBase
	case storepb.Engine_MYSQL:
		explainCalculator = getAffectedRowsCountForMysql
	}
	totalAffectedRows, err := calculateAffectedRows(ctx, changeSummary, dbMetadata, sqlDB, explainCalculator)
	if err != nil {
		return nil, err
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status: storepb.PlanCheckRunResult_Result_SUCCESS,
			Code:   common.Ok.Int32(),
			Title:  "OK",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					StatementTypes:   sqlTypes,
					AffectedRows:     int32(totalAffectedRows),
					ChangedResources: convertToChangedResources(dbMetadata, changeSummary.ResourceChanges),
				},
			},
		},
	}, nil
}

func reportForOracle(databaseName string, schemaName string, asts any, statement string, dbMetadata *model.DBSchema) ([]*storepb.PlanCheckRunResult_Result, error) {
	changeSummary, err := base.ExtractChangedResources(storepb.Engine_ORACLE, databaseName, schemaName, asts, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract changed resources")
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status: storepb.PlanCheckRunResult_Result_SUCCESS,
			Code:   common.Ok.Int32(),
			Title:  "OK",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					StatementTypes:   nil,
					AffectedRows:     0,
					ChangedResources: convertToChangedResources(dbMetadata, changeSummary.ResourceChanges),
				},
			},
		},
	}, nil
}

type getAffectedRowsFromExplain func(context.Context, *sql.DB, string) (int64, error)

func calculateAffectedRows(ctx context.Context, changeSummary *base.ChangeSummary, dbMetadata *model.DBSchema, sqlDB *sql.DB, explainCalculator getAffectedRowsFromExplain) (int64, error) {
	var totalAffectedRows int64
	// Count DMLs.
	for _, dml := range changeSummary.SampleDMLS {
		count, err := explainCalculator(ctx, sqlDB, dml)
		if err != nil {
			return 0, err
		}
		totalAffectedRows += count
	}
	if changeSummary.DMLCount > common.MaximumLintExplainSize {
		totalAffectedRows = int64((float64(totalAffectedRows) / common.MaximumLintExplainSize) * float64(changeSummary.DMLCount))
	}
	totalAffectedRows += int64(changeSummary.InsertCount)
	// Count affected rows by DDLs.
	for _, change := range changeSummary.ResourceChanges {
		if !change.AffectTable {
			continue
		}
		if dbMetadata == nil {
			continue
		}
		dbMeta := dbMetadata.GetDatabaseMetadata()
		if dbMeta == nil {
			continue
		}
		schemaMeta := dbMeta.GetSchema(change.Resource.Schema)
		if schemaMeta == nil {
			continue
		}
		tableMeta := schemaMeta.GetTable(change.Resource.Table)
		if tableMeta == nil {
			continue
		}
		totalAffectedRows += tableMeta.GetRowCount()
	}

	return totalAffectedRows, nil
}

func convertToChangedResources(dbMetadata *model.DBSchema, resourceChanges []*base.ResourceChange) *storepb.ChangedResources {
	meta := &storepb.ChangedResources{}
	// resourceChange is ordered by (db, schema, table)
	for _, resourceChange := range resourceChanges {
		resource := resourceChange.Resource
		if len(meta.Databases) == 0 || meta.Databases[len(meta.Databases)-1].Name != resource.Database {
			meta.Databases = append(meta.Databases, &storepb.ChangedResourceDatabase{Name: resource.Database})
		}
		database := meta.Databases[len(meta.Databases)-1]
		if len(database.Schemas) == 0 || database.Schemas[len(database.Schemas)-1].Name != resource.Schema {
			database.Schemas = append(database.Schemas, &storepb.ChangedResourceSchema{Name: resource.Schema})
		}
		schema := database.Schemas[len(database.Schemas)-1]
		var tableRows int64
		if dbMetadata != nil && dbMetadata.GetDatabaseMetadata() != nil && dbMetadata.GetDatabaseMetadata().GetSchema(resource.Schema) != nil && dbMetadata.GetDatabaseMetadata().GetSchema(resource.Schema).GetTable(resource.Table) != nil {
			tableRows = dbMetadata.GetDatabaseMetadata().GetSchema(resource.Schema).GetTable(resource.Table).GetRowCount()
		}
		var ranges []*storepb.Range
		for _, r := range resourceChange.Ranges {
			ranges = append(ranges, &storepb.Range{
				Start: int32(r.Start),
				End:   int32(r.End),
			})
		}
		schema.Tables = append(schema.Tables, &storepb.ChangedResourceTable{
			Name:      resource.Table,
			TableRows: tableRows,
			Ranges:    ranges,
		})
	}
	return meta
}
