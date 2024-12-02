package v1

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/alexmullins/zip"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mapperparser "github.com/bytebase/bytebase/backend/plugin/parser/mybatis/mapper"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	// defaultTimeout is the default timeout for query and admin execution.
	defaultTimeout = 10 * time.Minute

	backupDatabaseName       = "bbdataarchive"
	oracleBackupDatabaseName = "BBDATAARCHIVE"
)

var (
	queryNewACLSupportEngines = map[storepb.Engine]bool{
		storepb.Engine_MYSQL:    true,
		storepb.Engine_POSTGRES: true,
		storepb.Engine_ORACLE:   true,
		storepb.Engine_MSSQL:    true,
	}
)

// SQLService is the service for SQL.
type SQLService struct {
	v1pb.UnimplementedSQLServiceServer
	store          *store.Store
	sheetManager   *sheet.Manager
	schemaSyncer   *schemasync.Syncer
	dbFactory      *dbfactory.DBFactory
	licenseService enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewSQLService creates a SQLService.
func NewSQLService(
	store *store.Store,
	sheetManager *sheet.Manager,
	schemaSyncer *schemasync.Syncer,
	dbFactory *dbfactory.DBFactory,
	licenseService enterprise.LicenseService,
	profile *config.Profile,
	iamManager *iam.Manager,
) *SQLService {
	return &SQLService{
		store:          store,
		sheetManager:   sheetManager,
		schemaSyncer:   schemaSyncer,
		dbFactory:      dbFactory,
		licenseService: licenseService,
		profile:        profile,
		iamManager:     iamManager,
	}
}

// AdminExecute executes the SQL statement.
func (s *SQLService) AdminExecute(server v1pb.SQLService_AdminExecuteServer) error {
	ctx := server.Context()
	var driver db.Driver
	var conn *sql.Conn
	defer func() {
		if conn != nil {
			if err := conn.Close(); err != nil {
				slog.Warn("failed to close connection", log.BBError(err))
			}
		}
		if driver != nil {
			driver.Close(ctx)
		}
	}()
	for {
		request, err := server.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return status.Errorf(codes.Internal, "failed to receive request: %v", err)
		}

		user, instance, database, err := s.prepareRelatedMessage(ctx, request.Name)
		if err != nil {
			return err
		}

		// We only need to get the driver and connection once.
		if driver == nil {
			driver, err = s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
			if err != nil {
				return status.Errorf(codes.Internal, "failed to get database driver: %v", err)
			}
			sqlDB := driver.GetDB()
			if sqlDB != nil {
				conn, err = sqlDB.Conn(ctx)
				if err != nil {
					return status.Errorf(codes.Internal, "failed to get database connection: %v", err)
				}
			}
		}

		queryContext := db.QueryContext{OperatorEmail: user.Email}
		if request.Schema != nil {
			queryContext.Schema = *request.Schema
		}
		result, duration, queryErr := executeWithTimeout(ctx, driver, conn, request.Statement, request.Timeout, queryContext)

		if err := s.createQueryHistory(ctx, database, store.QueryHistoryTypeQuery, request.Statement, user.ID, duration, queryErr); err != nil {
			slog.Error("failed to post admin execute activity", log.BBError(err))
		}

		response := &v1pb.AdminExecuteResponse{}
		if queryErr != nil {
			response.Results = []*v1pb.QueryResult{
				{
					Error: queryErr.Error(),
				},
			}
		} else {
			response.Results = result
		}

		if err := server.Send(response); err != nil {
			return status.Errorf(codes.Internal, "failed to send response: %v", err)
		}
	}
}

func (s *SQLService) Query(ctx context.Context, request *v1pb.QueryRequest) (*v1pb.QueryResponse, error) {
	// Prepare related message.
	user, instance, database, err := s.prepareRelatedMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}

	statement := request.Statement
	// In Redshift datashare, Rewrite query used for parser.
	if database.DataShare {
		statement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", database.DatabaseName), "")
	}

	// Validate the request.
	// New query ACL experience.
	if !request.Explain && !queryNewACLSupportEngines[instance.Engine] {
		if err := validateQueryRequest(instance, statement); err != nil {
			return nil, err
		}
	}

	dataSource, err := checkAndGetDataSourceQueriable(ctx, s.store, database, request.DataSourceId)
	if err != nil {
		return nil, err
	}
	driver, err := s.dbFactory.GetDataSourceDriver(ctx, instance, dataSource, database.DatabaseName, database.DataShare, dataSource.Type == api.RO /* readOnly */, db.ConnectionContext{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get database driver: %v", err)
	}
	defer driver.Close(ctx)

	sqlDB := driver.GetDB()
	var conn *sql.Conn
	if sqlDB != nil {
		conn, err = sqlDB.Conn(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get database connection: %v", err)
		}
		defer conn.Close()
	}

	queryContext := db.QueryContext{
		Explain:       request.Explain,
		Limit:         int(request.Limit),
		OperatorEmail: user.Email,
		Option:        request.QueryOption,
	}
	if request.Schema != nil {
		queryContext.Schema = *request.Schema
	}
	results, spans, duration, queryErr := queryRetry(ctx, s.store, user, instance, database, driver, conn, statement, request.Timeout, queryContext, false, s.licenseService, s.accessCheck, s.schemaSyncer)

	// Update activity.
	if err = s.createQueryHistory(ctx, database, store.QueryHistoryTypeQuery, statement, user.ID, duration, queryErr); err != nil {
		return nil, err
	}
	if queryErr != nil {
		code := codes.Internal
		if status, ok := status.FromError(queryErr); ok && status.Code() != codes.OK && status.Code() != codes.Unknown {
			code = status.Code()
		}
		return nil, status.Error(code, queryErr.Error())
	}

	// AllowExport is a validate only check.
	checkErr := s.accessCheck(ctx, instance, database, user, spans, queryContext.Limit, request.Explain, true /* isExport */)
	allowExport := (checkErr == nil)

	response := &v1pb.QueryResponse{
		Results:     results,
		AllowExport: allowExport,
	}

	return response, nil
}

type accessCheckFunc func(context.Context, *store.InstanceMessage, *store.DatabaseMessage, *store.UserMessage, []*base.QuerySpan, int, bool /* isExplain */, bool /* isExport */) error

func extractSourceTable(comment string) (string, string, string, error) {
	pattern := `\((\w+),\s*(\w+)(?:,\s*(\w+))?\)`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(comment)

	if len(matches) == 3 || (len(matches) == 4 && matches[3] == "") {
		databaseName := matches[1]
		tableName := matches[2]
		return databaseName, "", tableName, nil
	} else if len(matches) == 4 {
		databaseName := matches[1]
		schemaName := matches[2]
		tableName := matches[3]
		return databaseName, schemaName, tableName, nil
	}

	return "", "", "", errors.Errorf("failed to extract source table from comment: %s", comment)
}

func getSchemaMetadata(engine storepb.Engine, dbSchema *model.DBSchema) *model.SchemaMetadata {
	switch engine {
	case storepb.Engine_POSTGRES:
		return dbSchema.GetDatabaseMetadata().GetSchema(backupDatabaseName)
	case storepb.Engine_MSSQL:
		return dbSchema.GetDatabaseMetadata().GetSchema("dbo")
	default:
		return dbSchema.GetDatabaseMetadata().GetSchema("")
	}
}

func replaceBackupTableWithSource(ctx context.Context, stores *store.Store, instance *store.InstanceMessage, database *store.DatabaseMessage, spans []*base.QuerySpan) error {
	switch instance.Engine {
	case storepb.Engine_POSTGRES:
		// Don't need to check the database name for postgres here.
		// We backup the table to the same database with bbdataarchive schema for Postgres.
	case storepb.Engine_ORACLE:
		if database.DatabaseName != oracleBackupDatabaseName {
			return nil
		}
	default:
		if database.DatabaseName != backupDatabaseName {
			return nil
		}
	}
	dbSchema, err := stores.GetDBSchema(ctx, database.UID)
	if err != nil {
		return err
	}
	schema := getSchemaMetadata(instance.Engine, dbSchema)
	if schema == nil {
		return nil
	}

	for _, span := range spans {
		span.SourceColumns = generateNewSourceColumnSet(instance.Engine, span.SourceColumns, schema)
		for _, result := range span.Results {
			result.SourceColumns = generateNewSourceColumnSet(instance.Engine, result.SourceColumns, schema)
		}
	}
	return nil
}

func generateNewSourceColumnSet(engine storepb.Engine, origin base.SourceColumnSet, schema *model.SchemaMetadata) base.SourceColumnSet {
	result := make(base.SourceColumnSet)
	for column := range origin {
		if isBackupTable(engine, column) {
			tableSchema := schema.GetTable(column.Table)
			if tableSchema == nil {
				result[column] = true
				continue
			}
			sourceDatabase, sourceSchema, sourceTable, err := extractSourceTable(tableSchema.GetTableComment())
			if err != nil {
				slog.Debug("failed to extract source table", log.BBError(err))
				result[column] = true
				continue
			}
			newColumn := generateNewColumn(engine, column, sourceDatabase, sourceSchema, sourceTable)
			result[newColumn] = true
		} else {
			result[column] = true
		}
	}
	return result
}

func generateNewColumn(engine storepb.Engine, column base.ColumnResource, database, schema, table string) base.ColumnResource {
	switch engine {
	case storepb.Engine_POSTGRES:
		return base.ColumnResource{
			Server:   column.Server,
			Database: column.Database,
			Schema:   database,
			Table:    table,
			Column:   column.Column,
		}
	default:
		return base.ColumnResource{
			Server:   column.Server,
			Database: database,
			Schema:   schema,
			Table:    table,
			Column:   column.Column,
		}
	}
}

func isBackupTable(engine storepb.Engine, column base.ColumnResource) bool {
	switch engine {
	case storepb.Engine_POSTGRES:
		return column.Schema == backupDatabaseName
	case storepb.Engine_ORACLE:
		return column.Database == oracleBackupDatabaseName
	default:
		return column.Database == backupDatabaseName
	}
}

func queryRetry(
	ctx context.Context,
	stores *store.Store,
	user *store.UserMessage,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	driver db.Driver,
	conn *sql.Conn,
	statement string,
	timeout *durationpb.Duration,
	queryContext db.QueryContext,
	isExport bool,
	licenseService enterprise.LicenseService,
	optionalAccessCheck accessCheckFunc,
	schemaSyncer *schemasync.Syncer,
) ([]*v1pb.QueryResult, []*base.QuerySpan, time.Duration, error) {
	var spans []*base.QuerySpan
	var err error
	if !queryContext.Explain {
		spans, err = base.GetQuerySpan(
			ctx,
			base.GetQuerySpanContext{
				InstanceID:                    instance.ResourceID,
				GetDatabaseMetadataFunc:       BuildGetDatabaseMetadataFunc(stores),
				ListDatabaseNamesFunc:         BuildListDatabaseNamesFunc(stores),
				GetLinkedDatabaseMetadataFunc: BuildGetLinkedDatabaseMetadataFunc(stores, instance.Engine),
			},
			instance.Engine,
			statement,
			database.DatabaseName,
			queryContext.Schema,
			store.IgnoreDatabaseAndTableCaseSensitive(instance),
		)
		if err != nil {
			return nil, nil, time.Duration(0), status.Errorf(codes.Internal, "failed to get query span: %v", err.Error())
		}
		// After replacing backup table with source, we can apply the original access check and mask sensitive data for backup table.
		// If err != nil, this function will return the original spans.
		if err := replaceBackupTableWithSource(ctx, stores, instance, database, spans); err != nil {
			slog.Debug("failed to replace backup table with source", log.BBError(err))
		}
		if optionalAccessCheck != nil {
			if err := optionalAccessCheck(ctx, instance, database, user, spans, queryContext.Limit, queryContext.Explain, isExport); err != nil {
				return nil, nil, time.Duration(0), err
			}
		}
	}

	results, duration, queryErr := executeWithTimeout(ctx, driver, conn, statement, timeout, queryContext)
	if queryErr != nil {
		return nil, nil, duration, queryErr
	}
	if queryContext.Explain {
		return results, nil, duration, nil
	}
	syncDatabaseMap := make(map[string]bool)
	for i, r := range results {
		if r.Error != "" {
			continue
		}
		if i < len(spans) && spans[i].NotFoundError != nil {
			for k := range spans[i].SourceColumns {
				syncDatabaseMap[k.Database] = true
			}
		}
	}

	// Sync database metadata.
	for accessDatabaseName := range syncDatabaseMap {
		d, err := stores.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &accessDatabaseName})
		if err != nil {
			return nil, nil, duration, err
		}
		if err := schemaSyncer.SyncDatabaseSchema(ctx, d, false /* force */); err != nil {
			return nil, nil, duration, errors.Wrapf(err, "failed to sync database schema for database %q", accessDatabaseName)
		}
	}

	// Retry getting query span.
	if len(syncDatabaseMap) > 0 {
		spans, err = base.GetQuerySpan(
			ctx,
			base.GetQuerySpanContext{
				InstanceID:                    instance.ResourceID,
				GetDatabaseMetadataFunc:       BuildGetDatabaseMetadataFunc(stores),
				ListDatabaseNamesFunc:         BuildListDatabaseNamesFunc(stores),
				GetLinkedDatabaseMetadataFunc: BuildGetLinkedDatabaseMetadataFunc(stores, instance.Engine),
			},
			instance.Engine,
			statement,
			database.DatabaseName,
			queryContext.Schema,
			store.IgnoreDatabaseAndTableCaseSensitive(instance),
		)
		if err != nil {
			return nil, nil, duration, status.Errorf(codes.Internal, "failed to get query span: %v", err.Error())
		}
		// After replacing backup table with source, we can apply the original access check and mask sensitive data for backup table.
		// If err != nil, this function will return the original spans.
		if err := replaceBackupTableWithSource(ctx, stores, instance, database, spans); err != nil {
			slog.Debug("failed to replace backup table with source", log.BBError(err))
		}
	}
	// The second query span should not tolerate any error, but we should retail the original error from database if possible.
	for i, result := range results {
		if i < len(spans) && result.Error == "" && spans[i].NotFoundError != nil {
			return nil, nil, duration, status.Errorf(codes.Internal, "failed to get query span: %v", spans[i].NotFoundError)
		}
	}

	if licenseService.IsFeatureEnabledForInstance(api.FeatureSensitiveData, instance) == nil && !queryContext.Explain {
		masker := NewQueryResultMasker(stores)
		if err := masker.MaskResults(ctx, spans, results, instance, storepb.MaskingExceptionPolicy_MaskingException_QUERY); err != nil {
			return nil, nil, duration, status.Error(codes.Internal, err.Error())
		}
	}
	return results, spans, duration, nil
}

func executeWithTimeout(ctx context.Context, driver db.Driver, conn *sql.Conn, statement string, timeout *durationpb.Duration, queryContext db.QueryContext) ([]*v1pb.QueryResult, time.Duration, error) {
	ctxTimeout := defaultTimeout
	if timeout != nil {
		ctxTimeout = timeout.AsDuration()
	}
	start := time.Now()
	ctx, cancelCtx := context.WithTimeout(ctx, ctxTimeout)
	defer cancelCtx()
	result, err := driver.QueryConn(ctx, conn, statement, queryContext)
	select {
	case <-ctx.Done():
		// canceled or timed out
		return nil, time.Since(start), errors.Errorf("timeout reached: %v", ctxTimeout)
	default:
		// So the select will not block
	}
	sanitizeResults(result)
	return result, time.Since(start), err
}

// Export exports the SQL query result.
func (s *SQLService) Export(ctx context.Context, request *v1pb.ExportRequest) (*v1pb.ExportResponse, error) {
	exportDataPolicy, err := s.store.GetDataExportPolicy(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get data export policy: %v", err)
	}
	if exportDataPolicy.Disable {
		return nil, status.Errorf(codes.PermissionDenied, "data export is not allowed")
	}
	// Prehandle export from issue.
	if strings.HasPrefix(request.Name, common.ProjectNamePrefix) {
		return s.doExportFromIssue(ctx, request.Name)
	}
	// Prepare related message.
	user, instance, database, err := s.prepareRelatedMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}

	statement := request.Statement
	// In Redshift datashare, Rewrite query used for parser.
	if database.DataShare {
		statement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", database.DatabaseName), "")
	}

	// Validate the request.
	// New query ACL experience.
	if instance.Engine != storepb.Engine_MYSQL {
		if err := validateQueryRequest(instance, statement); err != nil {
			return nil, err
		}
	}

	bytes, duration, exportErr := DoExport(ctx, s.store, s.dbFactory, s.licenseService, request, user, instance, database, s.accessCheck, s.schemaSyncer)

	if err := s.createQueryHistory(ctx, database, store.QueryHistoryTypeExport, statement, user.ID, duration, exportErr); err != nil {
		return nil, err
	}

	if exportErr != nil {
		return nil, status.Error(codes.Internal, exportErr.Error())
	}

	return &v1pb.ExportResponse{
		Content: bytes,
	}, nil
}

func (s *SQLService) doExportFromIssue(ctx context.Context, issueName string) (*v1pb.ExportResponse, error) {
	issueUID, err := common.GetIssueID(issueName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get issue ID: %v", err)
	}
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &issueUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue: %v", err)
	}
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	if user.ID != issue.Creator.ID {
		return nil, status.Errorf(codes.PermissionDenied, "only the issue creator can download")
	}
	if issue.PipelineUID == nil {
		return nil, status.Errorf(codes.InvalidArgument, "issue %s has no pipeline", issueName)
	}
	rollout, err := s.store.GetRollout(ctx, *issue.PipelineUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rollout: %v", err)
	}
	if rollout == nil {
		return nil, status.Errorf(codes.NotFound, "rollout %d not found", *issue.PipelineUID)
	}
	tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: &rollout.ID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get tasks: %v", err)
	}
	if len(tasks) != 1 {
		return nil, status.Errorf(codes.InvalidArgument, "issue %s has unmatched tasks", issueName)
	}
	task := tasks[0]
	taskRuns, err := s.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{TaskUID: &task.ID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get task run: %v", err)
	}
	if len(taskRuns) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "issue %s has no task run", issueName)
	}
	taskRun := taskRuns[len(taskRuns)-1]
	exportArchiveUID := int(taskRun.ResultProto.ExportArchiveUid)
	if exportArchiveUID == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "issue %s has no export archive", issueName)
	}
	exportArchive, err := s.store.GetExportArchive(ctx, &store.FindExportArchiveMessage{UID: &exportArchiveUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get export archive: %v", err)
	}
	if exportArchive == nil {
		return nil, status.Errorf(codes.NotFound, "export archive %d not found", exportArchiveUID)
	}
	// Delete the export archive after it's fetched.
	if err := s.store.DeleteExportArchive(ctx, exportArchiveUID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete export archive: %v", err)
	}
	return &v1pb.ExportResponse{
		Content: exportArchive.Bytes,
	}, nil
}

// DoExport does the export.
func DoExport(
	ctx context.Context,
	storeInstance *store.Store,
	dbFactory *dbfactory.DBFactory,
	licenseService enterprise.LicenseService,
	request *v1pb.ExportRequest,
	user *store.UserMessage,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	optionalAccessCheck accessCheckFunc,
	schemaSyncer *schemasync.Syncer,
) ([]byte, time.Duration, error) {
	dataSource, err := checkAndGetDataSourceQueriable(ctx, storeInstance, database, request.DataSourceId)
	if err != nil {
		return nil, 0, err
	}
	driver, err := dbFactory.GetDataSourceDriver(ctx, instance, dataSource, database.DatabaseName, database.DataShare, true /* readOnly */, db.ConnectionContext{})
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, "failed to get database driver: %v", err)
	}
	defer driver.Close(ctx)

	sqlDB := driver.GetDB()
	var conn *sql.Conn
	if sqlDB != nil {
		conn, err = sqlDB.Conn(ctx)
		if err != nil {
			return nil, 0, err
		}
		defer conn.Close()
	}
	queryContext := db.QueryContext{
		Limit:         int(request.Limit),
		OperatorEmail: user.Email,
	}
	results, spans, duration, queryErr := queryRetry(ctx, storeInstance, user, instance, database, driver, conn, request.Statement, nil /* timeDuration */, queryContext, true, licenseService, optionalAccessCheck, schemaSyncer)
	if queryErr != nil {
		return nil, duration, err
	}
	// only return the last result
	if len(results) > 1 {
		results = results[len(results)-1:]
	}
	if results[0].GetError() != "" {
		return nil, duration, errors.New(results[0].GetError())
	}

	if licenseService.IsFeatureEnabledForInstance(api.FeatureSensitiveData, instance) == nil {
		masker := NewQueryResultMasker(storeInstance)
		if err := masker.MaskResults(ctx, spans, results, instance, storepb.MaskingExceptionPolicy_MaskingException_EXPORT); err != nil {
			return nil, duration, err
		}
	}

	result := results[0]
	var content []byte
	switch request.Format {
	case v1pb.ExportFormat_CSV:
		content, err = exportCSV(result)
		if err != nil {
			return nil, duration, err
		}
	case v1pb.ExportFormat_JSON:
		content, err = exportJSON(result)
		if err != nil {
			return nil, duration, err
		}
	case v1pb.ExportFormat_SQL:
		resourceList, err := extractResourceList(ctx, storeInstance, instance.Engine, database.DatabaseName, request.Statement, instance)
		if err != nil {
			return nil, 0, status.Errorf(codes.InvalidArgument, "failed to extract resource list: %v", err)
		}
		statementPrefix, err := getSQLStatementPrefix(instance.Engine, resourceList, result.ColumnNames)
		if err != nil {
			return nil, 0, err
		}
		content, err = exportSQL(instance.Engine, statementPrefix, result)
		if err != nil {
			return nil, duration, err
		}
	case v1pb.ExportFormat_XLSX:
		content, err = exportXLSX(result)
		if err != nil {
			return nil, duration, err
		}
	default:
		return nil, duration, status.Errorf(codes.InvalidArgument, "unsupported export format: %s", request.Format.String())
	}

	encryptedBytes, err := doEncrypt(content, request)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to encrypt data")
	}
	return encryptedBytes, duration, nil
}

func doEncrypt(data []byte, request *v1pb.ExportRequest) ([]byte, error) {
	if request.Password == "" {
		return data, nil
	}
	var b bytes.Buffer
	fzip := io.Writer(&b)

	zipw := zip.NewWriter(fzip)
	defer zipw.Close()

	fh := &zip.FileHeader{
		Name:   fmt.Sprintf("export.%s", strings.ToLower(request.Format.String())),
		Method: zip.Deflate,
	}
	fh.ModifiedDate, fh.ModifiedTime = timeToMsDosTime(time.Now())
	fh.SetPassword(request.Password)
	writer, err := zipw.CreateHeader(fh)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create encrypt export file")
	}

	if _, err := io.Copy(writer, bytes.NewReader(data)); err != nil {
		return nil, errors.Wrapf(err, "failed to write export file")
	}
	if err := zipw.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close zip writer")
	}

	return b.Bytes(), nil
}

// timeToMsDosTime converts a time.Time to an MS-DOS date and time.
// this is a modified copy for gihub.com/alexmullins/zip/struct.go cause the package has a bug, it will convert the time to UTC time and drop the timezone.
func timeToMsDosTime(t time.Time) (uint16, uint16) {
	fDate := uint16(t.Day() + int(t.Month())<<5 + (t.Year()-1980)<<9)
	fTime := uint16(t.Second()/2 + t.Minute()<<5 + t.Hour()<<11)
	return fDate, fTime
}

func (s *SQLService) createQueryHistory(ctx context.Context, database *store.DatabaseMessage, queryType store.QueryHistoryType, statement string, userUID int, duration time.Duration, queryErr error) error {
	qh := &store.QueryHistoryMessage{
		CreatorUID: userUID,
		ProjectID:  database.ProjectID,
		Database:   common.FormatDatabase(database.InstanceID, database.DatabaseName),
		Statement:  statement,
		Type:       queryType,
		Payload: &storepb.QueryHistoryPayload{
			Error:    nil,
			Duration: durationpb.New(duration),
		},
	}
	if queryErr != nil {
		queryErrString := queryErr.Error()
		qh.Payload.Error = &queryErrString
	}

	if _, err := s.store.CreateQueryHistory(ctx, qh); err != nil {
		return status.Errorf(codes.Internal, "Failed to create export history with error: %v", err)
	}
	return nil
}

// SearchQueryHistories lists query histories.
func (s *SQLService) SearchQueryHistories(ctx context.Context, request *v1pb.SearchQueryHistoriesRequest) (*v1pb.SearchQueryHistoriesResponse, error) {
	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.PageToken,
		limit:   int(request.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}

	find := &store.FindQueryHistoryMessage{
		CreatorUID: &principalID,
		Limit:      &limitPlusOne,
		Offset:     &offset.offset,
	}

	filters, err := ParseFilter(request.Filter)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	for _, spec := range filters {
		if spec.Operator != ComparatorTypeEqual {
			return nil, status.Errorf(codes.InvalidArgument, `only support "=" operation for "%v" filter`, spec.Key)
		}
		switch spec.Key {
		case "database":
			database := spec.Value
			find.Database = &database
		case "instance":
			instance := spec.Value
			find.Instance = &instance
		case "type":
			historyType := store.QueryHistoryType(spec.Value)
			find.Type = &historyType
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter %s", spec.Key)
		}
	}

	historyList, err := s.store.ListQueryHistories(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list history: %v", err.Error())
	}

	nextPageToken := ""
	if len(historyList) == limitPlusOne {
		historyList = historyList[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal next page token, error: %v", err)
		}
	}

	resp := &v1pb.SearchQueryHistoriesResponse{
		NextPageToken: nextPageToken,
	}
	for _, history := range historyList {
		queryHistory, err := s.convertToV1QueryHistory(ctx, history)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert log entity, error: %v", err)
		}
		if queryHistory == nil {
			continue
		}
		resp.QueryHistories = append(resp.QueryHistories, queryHistory)
	}

	return resp, nil
}

func (s *SQLService) convertToV1QueryHistory(ctx context.Context, history *store.QueryHistoryMessage) (*v1pb.QueryHistory, error) {
	user, err := s.store.GetUserByID(ctx, history.CreatorUID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.Errorf("cannot found user with id %d", history.CreatorUID)
	}

	historyType := v1pb.QueryHistory_TYPE_UNSPECIFIED
	switch history.Type {
	case store.QueryHistoryTypeExport:
		historyType = v1pb.QueryHistory_EXPORT
	case store.QueryHistoryTypeQuery:
		historyType = v1pb.QueryHistory_QUERY
	}

	return &v1pb.QueryHistory{
		Name:       fmt.Sprintf("queryHistories/%d", history.UID),
		Statement:  history.Statement,
		Error:      history.Payload.Error,
		Database:   history.Database,
		Creator:    common.FormatUserEmail(user.Email),
		CreateTime: timestamppb.New(history.CreatedTime),
		Duration:   history.Payload.Duration,
		Type:       historyType,
	}, nil
}

func BuildGetLinkedDatabaseMetadataFunc(storeInstance *store.Store, engine storepb.Engine) base.GetLinkedDatabaseMetadataFunc {
	if engine != storepb.Engine_ORACLE {
		return nil
	}
	return func(ctx context.Context, instanceID string, linkedDatabaseName string, schemaName string) (string, string, *model.DatabaseMetadata, error) {
		// Find the linked database metadata.
		databases, err := storeInstance.ListDatabases(ctx, &store.FindDatabaseMessage{
			InstanceID: &instanceID,
		})
		if err != nil {
			return "", "", nil, err
		}
		var linkedMeta *model.LinkedDatabaseMetadata
		for _, database := range databases {
			meta, err := storeInstance.GetDBSchema(ctx, database.UID)
			if err != nil {
				return "", "", nil, err
			}
			if linkedMeta = meta.GetDatabaseMetadata().GetLinkedDatabase(linkedDatabaseName); linkedMeta != nil {
				break
			}
		}
		if linkedMeta == nil {
			return "", "", nil, nil
		}
		// Find the linked database in Bytebase.
		var linkedDatabase *store.DatabaseMessage
		databaseName := linkedMeta.GetUsername()
		if schemaName != "" {
			databaseName = schemaName
		}
		databaseList, err := storeInstance.ListDatabases(ctx, &store.FindDatabaseMessage{
			DatabaseName: &databaseName,
			Engine:       &engine,
		})
		if err != nil {
			return "", "", nil, err
		}
		for _, database := range databaseList {
			instanceMeta, err := storeInstance.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
			if err != nil {
				return "", "", nil, err
			}
			if instanceMeta != nil {
				for _, dataSource := range instanceMeta.DataSources {
					if strings.Contains(linkedMeta.GetHost(), dataSource.Host) {
						linkedDatabase = database
						break
					}
				}
				if linkedDatabase != nil {
					break
				}
			}
		}
		if linkedDatabase == nil {
			return "", "", nil, nil
		}
		// Get the linked database metadata.
		linkedDatabaseMetadata, err := storeInstance.GetDBSchema(ctx, linkedDatabase.UID)
		if err != nil {
			return "", "", nil, err
		}
		if linkedDatabaseMetadata == nil {
			return "", "", nil, nil
		}
		return linkedDatabase.InstanceID, linkedDatabaseName, linkedDatabaseMetadata.GetDatabaseMetadata(), nil
	}
}

func BuildGetDatabaseMetadataFunc(storeInstance *store.Store) base.GetDatabaseMetadataFunc {
	return func(ctx context.Context, instanceID, databaseName string) (string, *model.DatabaseMetadata, error) {
		database, err := storeInstance.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instanceID,
			DatabaseName: &databaseName,
		})
		if err != nil {
			return "", nil, err
		}
		if database == nil {
			return "", nil, nil
		}
		databaseMetadata, err := storeInstance.GetDBSchema(ctx, database.UID)
		if err != nil {
			return "", nil, err
		}
		if databaseMetadata == nil {
			return "", nil, nil
		}
		return databaseName, databaseMetadata.GetDatabaseMetadata(), nil
	}
}

func BuildListDatabaseNamesFunc(storeInstance *store.Store) base.ListDatabaseNamesFunc {
	return func(ctx context.Context, instanceID string) ([]string, error) {
		databases, err := storeInstance.ListDatabases(ctx, &store.FindDatabaseMessage{
			InstanceID: &instanceID,
		})
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(databases))
		for _, database := range databases {
			names = append(names, database.DatabaseName)
		}
		return names, nil
	}
}

func (s *SQLService) accessCheck(
	ctx context.Context,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	user *store.UserMessage,
	spans []*base.QuerySpan,
	limit int,
	isExplain bool,
	isExport bool) error {
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return err
	}
	if project == nil {
		return status.Errorf(codes.InvalidArgument, "project %q not found", database.ProjectID)
	}

	for _, span := range spans {
		// New query ACL experience.
		if queryNewACLSupportEngines[instance.Engine] {
			var permission iam.Permission
			switch span.Type {
			case base.QueryTypeUnknown:
				return status.Error(codes.PermissionDenied, "disallowed query type")
			case base.DDL:
				permission = iam.PermissionSQLDdl
			case base.DML:
				permission = iam.PermissionSQLDml
			case base.Explain:
				permission = iam.PermissionSQLExplain
			case base.SelectInfoSchema:
				permission = iam.PermissionSQLInfo
			case base.Select:
				// Conditional permission check below.
			}
			if isExplain {
				permission = iam.PermissionSQLExplain
			}
			if span.Type == base.DDL || span.Type == base.DML {
				if err := checkDataSourceQueryPolicy(ctx, s.store, database, span.Type); err != nil {
					return err
				}
			}
			if permission != "" {
				ok, err := s.iamManager.CheckPermission(ctx, permission, user, project.ResourceID)
				if err != nil {
					return err
				}
				if !ok {
					return status.Errorf(codes.PermissionDenied, "user %q does not have permission %q on project %q", user.Email, permission, project.ResourceID)
				}
			}
		}
		if span.Type == base.Select {
			for column := range span.SourceColumns {
				attributes := map[string]any{
					"request.time":      time.Now(),
					"request.row_limit": limit,
					"resource.database": common.FormatDatabase(instance.ResourceID, column.Database),
					"resource.schema":   column.Schema,
					"resource.table":    column.Table,
				}

				databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
					InstanceID:          &instance.ResourceID,
					DatabaseName:        &column.Database,
					IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
				})
				if err != nil {
					return err
				}
				if databaseMessage == nil {
					return status.Errorf(codes.InvalidArgument, "database %q not found", column.Database)
				}
				project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &databaseMessage.ProjectID})
				if err != nil {
					return err
				}
				if project == nil {
					return status.Errorf(codes.InvalidArgument, "project %q not found", databaseMessage.ProjectID)
				}

				workspacePolicy, err := s.store.GetWorkspaceIamPolicy(ctx)
				if err != nil {
					return status.Errorf(codes.Internal, "failed to get workspace iam policy, error: %v", err)
				}
				// Allow query databases across different projects.
				projectPolicy, err := s.store.GetProjectIamPolicy(ctx, project.UID)
				if err != nil {
					return status.Error(codes.Internal, err.Error())
				}

				ok, err := s.hasDatabaseAccessRights(ctx, user, []*storepb.IamPolicy{workspacePolicy.Policy, projectPolicy.Policy}, attributes, isExport)
				if err != nil {
					return status.Errorf(codes.Internal, "failed to check access control for database: %q, error %v", column.Database, err)
				}
				if !ok {
					resource := attributes["resource.database"]
					if schema, ok := attributes["resource.schema"]; ok && schema != "" {
						resource = fmt.Sprintf("%s/schemas/%s", resource, schema)
					}
					if table, ok := attributes["resource.table"]; ok && table != "" {
						resource = fmt.Sprintf("%s/tables/%s", resource, table)
					}
					return status.Errorf(
						codes.PermissionDenied,
						"permission denied to access resource: %s", resource,
					)
				}
			}
		}
	}

	return nil
}

// sanitizeResults sanitizes the strings in the results by replacing all the invalid UTF-8 characters with its hexadecimal representation.
func sanitizeResults(results []*v1pb.QueryResult) {
	for _, result := range results {
		for _, row := range result.GetRows() {
			for _, value := range row.GetValues() {
				if value != nil {
					if value, ok := value.Kind.(*v1pb.RowValue_StringValue); ok {
						value.StringValue = common.SanitizeUTF8String(value.StringValue)
					}
				}
			}
		}
	}
}

func (s *SQLService) prepareRelatedMessage(ctx context.Context, requestName string) (*store.UserMessage, *store.InstanceMessage, *store.DatabaseMessage, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, nil, nil, status.Error(codes.Internal, err.Error())
	}

	instanceID, databaseName, err := common.GetInstanceDatabaseID(requestName)
	if err != nil {
		return nil, nil, nil, status.Error(codes.InvalidArgument, err.Error())
	}

	find := &store.FindInstanceMessage{
		ResourceID: &instanceID,
	}
	instance, err := s.store.GetInstanceV2(ctx, find)
	if err != nil {
		return nil, nil, nil, status.Error(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, nil, nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instance.ResourceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, nil, nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
	}
	if database == nil {
		return nil, nil, nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	return user, instance, database, nil
}

func validateQueryRequest(instance *store.InstanceMessage, statement string) error {
	ok, _, err := base.ValidateSQLForEditor(instance.Engine, statement)
	if err != nil {
		syntaxErr, ok := err.(*base.SyntaxError)
		if ok {
			querySyntaxError, err := status.New(codes.InvalidArgument, err.Error()).WithDetails(
				&v1pb.PlanCheckRun_Result_SqlReviewReport{
					Line:   int32(syntaxErr.Line),
					Column: int32(syntaxErr.Column),
					Detail: syntaxErr.Message,
				},
			)
			if err != nil {
				return syntaxErr
			}
			return querySyntaxError.Err()
		}
		return err
	}
	if !ok {
		return nonSelectSQLError.Err()
	}
	return nil
}

func (s *SQLService) hasDatabaseAccessRights(ctx context.Context, user *store.UserMessage, iamPolicies []*storepb.IamPolicy, attributes map[string]any, isExport bool) (bool, error) {
	wantPermission := iam.PermissionSQLSelect
	if isExport {
		wantPermission = iam.PermissionSQLExport
	}

	bindings := utils.GetUserIAMPolicyBindings(ctx, s.store, user, iamPolicies...)
	for _, binding := range bindings {
		permissions, err := s.iamManager.GetPermissions(binding.Role)
		if err != nil {
			return false, errors.Wrapf(err, "failed to get permissions")
		}
		if !permissions[wantPermission] {
			continue
		}

		ok, err := evaluateQueryExportPolicyCondition(binding.Condition.GetExpression(), attributes)
		if err != nil {
			slog.Error("failed to evaluate condition", log.BBError(err), slog.String("condition", binding.Condition.GetExpression()))
			continue
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func (*SQLService) getUser(ctx context.Context) (*store.UserMessage, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	if user.MemberDeleted {
		return nil, status.Errorf(codes.PermissionDenied, "the user has been deactivated.")
	}
	return user, nil
}

func (s *SQLService) Check(ctx context.Context, request *v1pb.CheckRequest) (*v1pb.CheckResponse, error) {
	if len(request.Statement) > common.MaxSheetCheckSize {
		return nil, status.Errorf(codes.FailedPrecondition, "statement size exceeds maximum allowed size %dKB", common.MaxSheetCheckSize/1024)
	}

	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get instance, error: %v", err)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get database, error: %v", err)
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", request.Name)
	}

	var overideMetadata *storepb.DatabaseSchemaMetadata
	if request.Metadata != nil {
		overideMetadata, _, err = convertV1DatabaseMetadata(ctx, request.Metadata, nil /* optionalStores */)
		if err != nil {
			return nil, err
		}
	}
	_, adviceList, err := s.SQLReviewCheck(ctx, request.Statement, request.ChangeType, instance, database, overideMetadata)
	if err != nil {
		return nil, err
	}

	return &v1pb.CheckResponse{
		Advices: adviceList,
	}, nil
}

func GetClassificationByProject(ctx context.Context, stores *store.Store, projectID string) *storepb.DataClassificationSetting_DataClassificationConfig {
	project, err := stores.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		slog.Warn("failed to find project", slog.String("project", projectID), log.BBError(err))
		return nil
	}
	if project == nil {
		return nil
	}
	if project.DataClassificationConfigID == "" {
		return nil
	}
	classificationConfig, err := stores.GetDataClassificationConfigByID(ctx, project.DataClassificationConfigID)
	if err != nil {
		slog.Warn("failed to find classification", slog.String("project", projectID), slog.String("classification", project.DataClassificationConfigID), log.BBError(err))
		return nil
	}
	return classificationConfig
}

// SQLReviewCheck checks the SQL statement against the SQL review policy bind to given environment,
// against the database schema bind to the given database, if the overrideMetadata is provided,
// it will be used instead of fetching the database schema from the store.
func (s *SQLService) SQLReviewCheck(
	ctx context.Context,
	statement string,
	changeType v1pb.CheckRequest_ChangeType,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	overrideMetadata *storepb.DatabaseSchemaMetadata,
) (storepb.Advice_Status, []*v1pb.Advice, error) {
	if !isSQLReviewSupported(instance.Engine) || database == nil {
		return storepb.Advice_SUCCESS, nil, nil
	}

	dbMetadata := overrideMetadata
	if dbMetadata == nil {
		dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to fetch database schema for database %v", database.UID)
		}
		if dbSchema == nil {
			if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
				return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to sync database schema for database %v", database.UID)
			}
			dbSchema, err = s.store.GetDBSchema(ctx, database.UID)
			if err != nil {
				return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to fetch database schema for database %v", database.UID)
			}
			if dbSchema == nil {
				return storepb.Advice_ERROR, nil, errors.Wrapf(err, "cannot found schema for database %v", database.UID)
			}
		}
		dbMetadata = dbSchema.GetMetadata()
	}

	catalog, err := catalog.NewCatalog(ctx, s.store, database.UID, instance.Engine, store.IgnoreDatabaseAndTableCaseSensitive(instance), overrideMetadata)
	if err != nil {
		return storepb.Advice_ERROR, nil, status.Errorf(codes.Internal, "failed to create a catalog: %v", err)
	}

	useDatabaseOwner, err := getUseDatabaseOwner(ctx, s.store, instance, database, changeType)
	if err != nil {
		return storepb.Advice_ERROR, nil, status.Errorf(codes.Internal, "failed to get use database owner: %v", err)
	}
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{UseDatabaseOwner: useDatabaseOwner})
	if err != nil {
		return storepb.Advice_ERROR, nil, status.Errorf(codes.Internal, "failed to get database driver: %v", err)
	}
	defer driver.Close(ctx)
	connection := driver.GetDB()

	classificationConfig := GetClassificationByProject(ctx, s.store, database.ProjectID)
	context := advisor.SQLReviewCheckContext{
		Charset:                  dbMetadata.CharacterSet,
		Collation:                dbMetadata.Collation,
		ChangeType:               convertChangeType(changeType),
		DBSchema:                 dbMetadata,
		DbType:                   instance.Engine,
		Catalog:                  catalog,
		Driver:                   connection,
		Context:                  ctx,
		CurrentDatabase:          database.DatabaseName,
		ClassificationConfig:     classificationConfig,
		UsePostgresDatabaseOwner: useDatabaseOwner,
		ListDatabaseNamesFunc:    BuildListDatabaseNamesFunc(s.store),
		InstanceID:               instance.ResourceID,
	}

	reviewConfig, err := s.store.GetReviewConfigForDatabase(ctx, database)
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			// Continue to check the builtin rules.
			reviewConfig = &storepb.ReviewConfigPayload{}
		} else {
			return storepb.Advice_ERROR, nil, status.Errorf(codes.Internal, "failed to get SQL review policy with error: %v", err)
		}
	}

	res, err := advisor.SQLReviewCheck(s.sheetManager, statement, reviewConfig.SqlReviewRules, context)
	if err != nil {
		return storepb.Advice_ERROR, nil, status.Errorf(codes.Internal, "failed to exec SQL review with error: %v", err)
	}

	adviceLevel := storepb.Advice_SUCCESS
	var advices []*v1pb.Advice
	for _, advice := range res {
		switch advice.Status {
		case storepb.Advice_WARNING:
			if adviceLevel != storepb.Advice_ERROR {
				adviceLevel = storepb.Advice_WARNING
			}
		case storepb.Advice_ERROR:
			adviceLevel = storepb.Advice_ERROR
		case storepb.Advice_SUCCESS, storepb.Advice_STATUS_UNSPECIFIED:
			continue
		}

		advices = append(advices, convertToV1Advice(advice))
	}

	return adviceLevel, advices, nil
}

func getUseDatabaseOwner(ctx context.Context, stores *store.Store, instance *store.InstanceMessage, database *store.DatabaseMessage, changeType v1pb.CheckRequest_ChangeType) (bool, error) {
	if instance.Engine != storepb.Engine_POSTGRES || changeType == v1pb.CheckRequest_SQL_EDITOR {
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

func convertToV1Advice(advice *storepb.Advice) *v1pb.Advice {
	return &v1pb.Advice{
		Status:        convertAdviceStatus(advice.Status),
		Code:          int32(advice.Code),
		Title:         advice.Title,
		Content:       advice.Content,
		Line:          int32(advice.GetStartPosition().GetLine()),
		Column:        int32(advice.GetStartPosition().GetColumn()),
		Detail:        advice.Detail,
		StartPosition: convertToPosition(advice.StartPosition),
		EndPosition:   convertToPosition(advice.EndPosition),
	}
}

func convertAdviceStatus(status storepb.Advice_Status) v1pb.Advice_Status {
	switch status {
	case storepb.Advice_SUCCESS:
		return v1pb.Advice_SUCCESS
	case storepb.Advice_WARNING:
		return v1pb.Advice_WARNING
	case storepb.Advice_ERROR:
		return v1pb.Advice_ERROR
	default:
		return v1pb.Advice_STATUS_UNSPECIFIED
	}
}

// ParseMyBatisMapper parses a MyBatis mapper XML file and returns the multi-SQL statements.
func (*SQLService) ParseMyBatisMapper(_ context.Context, request *v1pb.ParseMyBatisMapperRequest) (*v1pb.ParseMyBatisMapperResponse, error) {
	content := string(request.Content)

	parser := mapperparser.NewParser(content)
	node, err := parser.Parse()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse mybatis mapper: %v", err)
	}

	var stringsBuilder strings.Builder
	if err := node.RestoreSQL(parser.NewRestoreContext().WithRestoreDataNodePlaceholder("@1"), &stringsBuilder); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to restore mybatis mapper: %v", err)
	}

	statement := stringsBuilder.String()
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_MYSQL, statement)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to split mybatis mapper: %v", err)
	}

	var results []string
	for _, sql := range singleSQLs {
		if sql.Empty {
			continue
		}
		results = append(results, sql.Text)
	}

	return &v1pb.ParseMyBatisMapperResponse{
		Statements: results,
	}, nil
}

// StringifyMetadata returns the stringified schema of the given metadata.
func (*SQLService) StringifyMetadata(ctx context.Context, request *v1pb.StringifyMetadataRequest) (*v1pb.StringifyMetadataResponse, error) {
	switch request.Engine {
	case v1pb.Engine_MYSQL, v1pb.Engine_OCEANBASE, v1pb.Engine_POSTGRES, v1pb.Engine_TIDB, v1pb.Engine_ORACLE:
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported engine: %v", request.Engine)
	}

	if request.Metadata == nil {
		return nil, status.Errorf(codes.InvalidArgument, "metadata is required")
	}
	storeSchemaMetadata, config, err := convertV1DatabaseMetadata(ctx, request.Metadata, nil /* optionalStores */)
	if err != nil {
		return nil, err
	}

	if !request.ClassificationFromConfig {
		sanitizeCommentForSchemaMetadata(storeSchemaMetadata, model.NewDatabaseConfig(config), request.ClassificationFromConfig)
	}

	if request.Engine == v1pb.Engine_MYSQL && isSingleTable(storeSchemaMetadata) {
		table := storeSchemaMetadata.Schemas[0].Tables[0]
		schema, err := schema.StringifyTable(storepb.Engine(request.Engine), table)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to stringify table: %v", err)
		}
		return &v1pb.StringifyMetadataResponse{
			Schema: schema,
		}, nil
	}

	defaultSchema := extractDefaultSchemaForOracleBranch(storepb.Engine(request.Engine), storeSchemaMetadata)
	schema, err := schema.GetDesignSchema(storepb.Engine(request.Engine), defaultSchema, storeSchemaMetadata)
	if err != nil {
		return nil, err
	}

	if request.Engine == v1pb.Engine_ORACLE {
		schema, err = appendComments(schema, storeSchemaMetadata)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to append comments: %v", err)
		}
	}

	return &v1pb.StringifyMetadataResponse{
		Schema: schema,
	}, nil
}

func appendComments(schema string, storeSchemaMetadata *storepb.DatabaseSchemaMetadata) (string, error) {
	if !isSingleTable(storeSchemaMetadata) {
		return schema, nil
	}

	schemaName := storeSchemaMetadata.Schemas[0].Name
	table := storeSchemaMetadata.Schemas[0].Tables[0]
	// Append comments to the schema.
	comments, err := getComments(schemaName, table)
	if err != nil {
		return "", err
	}
	return schema + comments, nil
}

func getComments(schemaName string, table *storepb.TableMetadata) (string, error) {
	var buf strings.Builder
	if table.Comment != "" {
		if _, err := fmt.Fprintf(&buf, "COMMENT ON TABLE \"%s\".\"%s\" IS '%s';\n", schemaName, table.Name, table.Comment); err != nil {
			return "", err
		}
	}
	for _, column := range table.Columns {
		if column.Comment != "" {
			if _, err := fmt.Fprintf(&buf, "COMMENT ON COLUMN \"%s\".\"%s\".\"%s\" IS '%s';\n", schemaName, table.Name, column.Name, column.Comment); err != nil {
				return "", err
			}
		}
	}
	return buf.String(), nil
}

func isSingleTable(storeSchemaMetadata *storepb.DatabaseSchemaMetadata) bool {
	if len(storeSchemaMetadata.Schemas) != 1 {
		return false
	}

	if len(storeSchemaMetadata.Schemas[0].Tables) != 1 {
		return false
	}

	if len(storeSchemaMetadata.Schemas[0].ExternalTables)+
		len(storeSchemaMetadata.Schemas[0].Views)+
		len(storeSchemaMetadata.Schemas[0].MaterializedViews)+
		len(storeSchemaMetadata.Schemas[0].Functions)+
		len(storeSchemaMetadata.Schemas[0].Procedures)+
		len(storeSchemaMetadata.Schemas[0].Sequences)+
		len(storeSchemaMetadata.Schemas[0].Streams)+
		len(storeSchemaMetadata.Schemas[0].Tasks) != 0 {
		return false
	}

	return true
}

// Pretty returns pretty format SDL.
func (*SQLService) Pretty(_ context.Context, request *v1pb.PrettyRequest) (*v1pb.PrettyResponse, error) {
	engine := convertEngine(request.Engine)
	if _, err := transform.CheckFormat(engine, request.ExpectedSchema); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "User SDL is not SDL format: %s", err.Error())
	}
	if _, err := transform.CheckFormat(engine, request.CurrentSchema); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Dumped SDL is not SDL format: %s", err.Error())
	}

	prettyExpectedSchema, err := transform.SchemaTransform(engine, request.ExpectedSchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to transform user SDL: %s", err.Error())
	}
	prettyCurrentSchema, err := transform.Normalize(engine, request.CurrentSchema, prettyExpectedSchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to normalize dumped SDL: %s", err.Error())
	}

	return &v1pb.PrettyResponse{
		CurrentSchema:  prettyCurrentSchema,
		ExpectedSchema: prettyExpectedSchema,
	}, nil
}

func getOffsetAndOriginTable(backupTable string) (int, string, error) {
	if backupTable == "" {
		return 0, "", nil
	}
	parts := strings.Split(backupTable, "_")
	if len(parts) < 4 {
		return 0, "", status.Errorf(codes.InvalidArgument, "invalid backup table format: %s", backupTable)
	}
	offset, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, "", status.Errorf(codes.InvalidArgument, "invalid offset: %s", parts[0])
	}
	return offset, strings.Join(parts[3:], "_"), nil
}

func checkAndGetDataSourceQueriable(ctx context.Context, storeInstance *store.Store, database *store.DatabaseMessage, dataSourceID string) (*store.DataSourceMessage, error) {
	if dataSourceID == "" {
		readOnlyDataSources, err := storeInstance.ListDataSourcesV2(ctx, &store.FindDataSourceMessage{
			InstanceID: &database.InstanceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list data sources: %v", err)
		}
		if len(readOnlyDataSources) == 0 {
			return nil, status.Errorf(codes.NotFound, "no data source found")
		}
		return readOnlyDataSources[0], nil
	}

	dataSource, err := storeInstance.GetDataSource(ctx, &store.FindDataSourceMessage{
		InstanceID: &database.InstanceID,
		Name:       &dataSourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get data source: %v", err)
	}
	if dataSource == nil {
		return nil, status.Errorf(codes.NotFound, "data source %q not found", dataSourceID)
	}

	// Always allow non-admin data source.
	if dataSource.Type != api.Admin {
		return dataSource, nil
	}

	var envAdminDataSourceRestriction, projectAdminDataSourceRestriction v1pb.DataSourceQueryPolicy_Restriction
	environment, err := storeInstance.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EffectiveEnvironmentID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get environment")
	}
	if environment == nil {
		return nil, errors.Errorf("environment %q not found", database.EffectiveEnvironmentID)
	}
	dataSourceQueryPolicyType := api.PolicyTypeDataSourceQuery
	environmentResourceType := api.PolicyResourceTypeEnvironment
	projectResourceType := api.PolicyResourceTypeProject
	environmentPolicy, err := storeInstance.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &environmentResourceType,
		ResourceUID:  &environment.UID,
		Type:         &dataSourceQueryPolicyType,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get policy")
	}
	if environmentPolicy != nil {
		envPayload, err := convertToV1PBDataSourceQueryPolicy(environmentPolicy.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert policy payload")
		}
		envAdminDataSourceRestriction = envPayload.DataSourceQueryPolicy.GetAdminDataSourceRestriction()
	}

	project, err := storeInstance.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return nil, errors.Errorf("project %q not found", database.ProjectID)
	}
	projectPolicy, err := storeInstance.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &projectResourceType,
		ResourceUID:  &project.UID,
		Type:         &dataSourceQueryPolicyType,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get policy")
	}
	if projectPolicy != nil {
		projectPayload, err := convertToV1PBDataSourceQueryPolicy(projectPolicy.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert policy payload")
		}
		projectAdminDataSourceRestriction = projectPayload.DataSourceQueryPolicy.GetAdminDataSourceRestriction()
	}

	// If any of the policy is DISALLOW, then return false.
	if envAdminDataSourceRestriction == v1pb.DataSourceQueryPolicy_DISALLOW || projectAdminDataSourceRestriction == v1pb.DataSourceQueryPolicy_DISALLOW {
		return nil, status.Errorf(codes.PermissionDenied, "data source %q is not queryable", dataSourceID)
	} else if envAdminDataSourceRestriction == v1pb.DataSourceQueryPolicy_FALLBACK || projectAdminDataSourceRestriction == v1pb.DataSourceQueryPolicy_FALLBACK {
		readOnlyDataSourceType := api.RO
		readOnlyDataSources, err := storeInstance.ListDataSourcesV2(ctx, &store.FindDataSourceMessage{
			InstanceID: &database.InstanceID,
			Type:       &readOnlyDataSourceType,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list read-only data sources")
		}
		// If there is any read-only data source, then return false.
		if len(readOnlyDataSources) > 0 {
			return nil, status.Errorf(codes.PermissionDenied, "data source %q is not queryable", dataSourceID)
		}
	}

	return dataSource, nil
}

func checkDataSourceQueryPolicy(ctx context.Context, storeInstance *store.Store, database *store.DatabaseMessage, statementTp base.QueryType) error {
	environment, err := storeInstance.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
		ResourceID: &database.EffectiveEnvironmentID,
	})
	if err != nil {
		return err
	}
	if environment == nil {
		return status.Errorf(codes.NotFound, "environment %q not found", database.EffectiveEnvironmentID)
	}
	resourceType := api.PolicyResourceTypeEnvironment
	policyType := api.PolicyTypeDataSourceQuery
	dataSourceQueryPolicy, err := storeInstance.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceUID:  &environment.UID,
		ResourceType: &resourceType,
		Type:         &policyType,
	})
	if err != nil {
		return err
	}
	if dataSourceQueryPolicy != nil {
		policy := &v1pb.DataSourceQueryPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(dataSourceQueryPolicy.Payload), policy); err != nil {
			return status.Errorf(codes.Internal, "failed to unmarshal data source query policy payload")
		}
		switch statementTp {
		case base.DDL:
			if policy.DisallowDdl {
				return status.Errorf(codes.PermissionDenied, "disallow execute DDL statement in environment %q", environment.Title)
			}
		case base.DML:
			if policy.DisallowDml {
				return status.Errorf(codes.PermissionDenied, "disallow execute DML statement in environment %q", environment.Title)
			}
		}
	}
	return nil
}
