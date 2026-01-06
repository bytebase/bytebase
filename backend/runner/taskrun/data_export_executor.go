package taskrun

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/alexmullins/zip"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/export"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
)

// NewDataExportExecutor creates a data export task executor.
func NewDataExportExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, license *enterprise.LicenseService) Executor {
	return &DataExportExecutor{
		store:     store,
		dbFactory: dbFactory,
		license:   license,
	}
}

// DataExportExecutor is the data export task executor.
type DataExportExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
	license   *enterprise.LicenseService
}

// RunOnce will run the data export task executor once.
func (exec *DataExportExecutor) RunOnce(ctx context.Context, _ context.Context, task *store.TaskMessage, _ int) (*storepb.TaskRunResult, error) {
	issue, err := exec.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &task.PlanID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get issue")
	}

	// Get plan to retrieve export format from spec
	plan, err := exec.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan")
	}
	if plan == nil {
		return nil, errors.Errorf("plan not found")
	}

	// For export data plans, there is always exactly one spec
	if len(plan.Config.Specs) == 0 {
		return nil, errors.Errorf("plan has no specs")
	}
	exportConfig := plan.Config.Specs[0].GetExportDataConfig()
	if exportConfig == nil {
		return nil, errors.Errorf("spec does not contain export data config")
	}

	database, err := exec.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName, ShowDeleted: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}
	instance, err := exec.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID, ShowDeleted: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance")
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found")
	}

	sheet, err := exec.store.GetSheetFull(ctx, task.Payload.GetSheetSha256())
	if err != nil {
		return nil, err
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet not found: %s", task.Payload.GetSheetSha256())
	}
	statement := sheet.Statement

	dataSource := apiv1.GetQueriableDataSource(instance)
	creatorUser, err := exec.store.GetUserByEmail(ctx, issue.CreatorEmail)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get creator user for issue %d", issue.UID)
	}
	if creatorUser == nil {
		return nil, errors.Errorf("creator user not found for issue %d", issue.UID)
	}

	// Execute the export without masking.
	// For approved DATABASE_EXPORT tasks, the approval itself authorizes access to the data.
	bytes, exportErr := exec.executeExport(ctx, instance, database, dataSource, statement, exportConfig.Format, creatorUser)
	if exportErr != nil {
		return nil, errors.Wrap(exportErr, "failed to export data")
	}

	exportArchive, err := exec.store.CreateExportArchive(ctx, &store.ExportArchiveMessage{
		Bytes: bytes,
		Payload: &storepb.ExportArchivePayload{
			FileFormat: exportConfig.Format,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create export archive")
	}

	return &storepb.TaskRunResult{
		ExportArchiveUid: int32(exportArchive.UID),
	}, nil
}

// executeExport performs the actual export without applying any masking.
// This is used for approved DATABASE_EXPORT tasks where the approval itself
// authorizes access to the data.
func (exec *DataExportExecutor) executeExport(
	ctx context.Context,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	dataSource *storepb.DataSource,
	statement string,
	format storepb.ExportFormat,
	user *store.UserMessage,
) ([]byte, error) {
	if dataSource == nil {
		return nil, errors.Errorf("cannot find valid data source")
	}

	// 1. Get driver and connection
	driver, err := exec.dbFactory.GetDataSourceDriver(ctx, instance, dataSource, db.ConnectionContext{
		DatabaseName: database.DatabaseName,
		DataShare:    database.Metadata.GetDatashare(),
		ReadOnly:     true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database driver")
	}
	defer driver.Close(ctx)

	sqlDB := driver.GetDB()
	var conn *sql.Conn
	if sqlDB != nil {
		conn, err = sqlDB.Conn(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get database connection")
		}
		defer conn.Close()
	}

	// 2. Get query restrictions from workspace policy
	queryRestriction := exec.getEffectiveQueryDataPolicy(ctx, database.ProjectID)

	// 3. Build query context with limits
	queryContext := db.QueryContext{
		Limit:                int(queryRestriction.MaximumResultRows),
		OperatorEmail:        user.Email,
		MaximumSQLResultSize: queryRestriction.MaximumResultSize,
	}
	if queryRestriction.MaxQueryTimeoutInSeconds > 0 {
		queryContext.Timeout = &durationpb.Duration{
			Seconds: queryRestriction.MaxQueryTimeoutInSeconds,
		}
	}

	// 4. Execute query with timeout
	queryCtx := ctx
	if queryContext.Timeout != nil {
		timeout := queryContext.Timeout.AsDuration()
		slog.Debug("create query context with timeout", slog.Duration("timeout", timeout))
		newCtx, cancelCtx := context.WithTimeout(ctx, timeout)
		defer cancelCtx()
		queryCtx = newCtx
	}

	start := time.Now()
	results, err := driver.QueryConn(queryCtx, conn, statement, queryContext)
	select {
	case <-queryCtx.Done():
		// canceled or timed out
		return nil, errors.Errorf("timeout reached: %v", queryContext.Timeout.AsDuration())
	default:
		// So the select will not block
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query")
	}
	slog.Debug("execute success", slog.String("instance", instance.ResourceID), slog.String("statement", statement), slog.Duration("duration", time.Since(start)))

	// 5. Format and zip results (NO MASKING)
	return exec.formatAndZipResults(ctx, results, instance, database, format, statement)
}

// getEffectiveQueryDataPolicy gets the effective query data policy for a project.
func (exec *DataExportExecutor) getEffectiveQueryDataPolicy(
	ctx context.Context,
	projectID string,
) *store.EffectiveQueryDataPolicy {
	value := &store.EffectiveQueryDataPolicy{
		MaximumResultSize: common.DefaultMaximumSQLResultSize,
		MaximumResultRows: math.MaxInt32,
	}
	if err := exec.license.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_QUERY_POLICY); err == nil {
		policy, err := exec.store.GetEffectiveQueryDataPolicy(ctx, common.FormatProject(projectID))
		if err != nil {
			slog.Error("failed to get the query data policy", log.BBError(err))
			return value
		}
		value = policy
	}
	if value.MaximumResultRows == math.MaxInt32 {
		value.MaximumResultRows = 0
	}
	return value
}

// formatAndZipResults formats query results and packages them into a ZIP archive.
func (exec *DataExportExecutor) formatAndZipResults(
	ctx context.Context,
	results []*v1pb.QueryResult,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	format storepb.ExportFormat,
	statement string,
) ([]byte, error) {
	var buf bytes.Buffer
	zipw := zip.NewWriter(&buf)

	exportCount := 0
	for i, result := range results {
		if result.GetError() != "" {
			logExportError(database, "failed to query result", errors.New(result.GetError()))
			continue
		}

		if err := exec.exportResultToZip(ctx, zipw, instance, database, result, format, statement, i+1); err != nil {
			logExportError(database, "failed to export result to zip", err)
			continue
		}

		exportCount++
		// Help GC by clearing the result data we've already processed
		result.Rows = nil
	}

	if exportCount == 0 {
		return nil, errors.Errorf("empty export data for database %s", database.DatabaseName)
	}

	if err := zipw.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close zip writer")
	}

	return buf.Bytes(), nil
}

// exportResultToZip exports a single query result to the ZIP archive.
// It writes both the SQL statement and the formatted result data.
func (exec *DataExportExecutor) exportResultToZip(
	ctx context.Context,
	zipw *zip.Writer,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	result *v1pb.QueryResult,
	format storepb.ExportFormat,
	statement string,
	statementNumber int,
) error {
	baseFilename := fmt.Sprintf("%s/%s/statement-%d", database.InstanceID, database.DatabaseName, statementNumber)

	// Write statement file
	statementFilename := fmt.Sprintf("%s.sql", baseFilename)
	if err := export.WriteZipEntry(zipw, statementFilename, []byte(statement), ""); err != nil {
		return errors.Wrap(err, "failed to write statement")
	}

	// Write result file by streaming directly to ZIP
	resultExt := strings.ToLower(v1pb.ExportFormat(format).String())
	resultFilename := fmt.Sprintf("%s.result.%s", baseFilename, resultExt)
	if err := exec.formatExportToZip(ctx, zipw, resultFilename, instance, database, result, format); err != nil {
		return errors.Wrap(err, "failed to write formatted result")
	}

	return nil
}

// formatExportToZip formats query results and writes them directly to a ZIP entry.
// This function streams the formatted data to minimize memory usage.
func (exec *DataExportExecutor) formatExportToZip(
	ctx context.Context,
	zipw *zip.Writer,
	filename string,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	result *v1pb.QueryResult,
	format storepb.ExportFormat,
) error {
	writer, err := export.CreateZipWriter(zipw, filename, "")
	if err != nil {
		return err
	}

	switch v1pb.ExportFormat(format) {
	case v1pb.ExportFormat_CSV:
		return export.CSVToWriter(writer, result)
	case v1pb.ExportFormat_JSON:
		return export.JSONToWriter(writer, result)
	case v1pb.ExportFormat_SQL:
		return exec.exportSQLWithContext(ctx, writer, instance, database, result)
	case v1pb.ExportFormat_XLSX:
		return export.XLSXToWriter(writer, result)
	default:
		return errors.Errorf("unsupported export format: %s", v1pb.ExportFormat(format).String())
	}
}

// exportSQLWithContext exports SQL INSERT statements with proper context.
func (exec *DataExportExecutor) exportSQLWithContext(
	ctx context.Context,
	w io.Writer,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	result *v1pb.QueryResult,
) error {
	resourceList, err := export.GetResources(
		ctx,
		exec.store,
		instance.Metadata.GetEngine(),
		database.DatabaseName,
		result.Statement,
		instance,
		apiv1.BuildGetDatabaseMetadataFunc(exec.store),
		apiv1.BuildListDatabaseNamesFunc(exec.store),
		apiv1.BuildGetLinkedDatabaseMetadataFunc(exec.store, instance.Metadata.GetEngine()),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to extract resource list")
	}
	statementPrefix, err := export.SQLStatementPrefix(instance.Metadata.GetEngine(), resourceList, result.ColumnNames)
	if err != nil {
		return err
	}
	return export.SQLToWriter(w, instance.Metadata.GetEngine(), statementPrefix, result)
}

// logExportError logs export-related errors with consistent database context.
func logExportError(database *store.DatabaseMessage, message string, err error) {
	slog.Error(message,
		log.BBError(err),
		slog.String("instance", database.InstanceID),
		slog.String("database", database.DatabaseName),
		slog.String("project", database.ProjectID),
	)
}
