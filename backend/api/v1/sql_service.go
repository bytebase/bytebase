package v1

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"log/slog"

	"connectrpc.com/connect"
	"github.com/alexmullins/zip"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/export"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
)

// SQLService is the service for SQL.
type SQLService struct {
	v1connect.UnimplementedSQLServiceHandler
	store          *store.Store
	sheetManager   *sheet.Manager
	schemaSyncer   *schemasync.Syncer
	dbFactory      *dbfactory.DBFactory
	licenseService *enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewSQLService creates a SQLService.
func NewSQLService(
	store *store.Store,
	sheetManager *sheet.Manager,
	schemaSyncer *schemasync.Syncer,
	dbFactory *dbfactory.DBFactory,
	licenseService *enterprise.LicenseService,
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
func (s *SQLService) AdminExecute(ctx context.Context, stream *connect.BidiStream[v1pb.AdminExecuteRequest, v1pb.AdminExecuteResponse]) error {
	var driver db.Driver
	var conn *sql.Conn
	var connectionName string

	clean := func() {
		if conn != nil {
			if err := conn.Close(); err != nil {
				slog.Warn("failed to close connection", log.BBError(err))
			}
		}
		if driver != nil {
			driver.Close(ctx)
		}
	}
	defer clean()
	for {
		request, err := stream.Receive()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return connect.NewError(connect.CodeInternal, errors.Errorf("failed to receive request: %v", err))
		}

		user, instance, database, err := s.prepareRelatedMessage(ctx, request.Name)
		if err != nil {
			return err
		}

		// We only need to get the driver and connection once.
		if driver == nil || connectionName != request.Name {
			clean()
			connectionName = request.Name
			driver, err = s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
			if err != nil {
				return connect.NewError(connect.CodeInternal, errors.Errorf("failed to get database driver: %v", err))
			}
			sqlDB := driver.GetDB()
			if sqlDB != nil {
				conn, err = sqlDB.Conn(ctx)
				if err != nil {
					return connect.NewError(connect.CodeInternal, errors.Errorf("failed to get database connection: %v", err))
				}
			}
		}

		queryRestriction := getMaximumSQLResultLimit(ctx, s.store, s.licenseService, 0)
		queryContext := db.QueryContext{
			OperatorEmail:        user.Email,
			Container:            request.GetContainer(),
			MaximumSQLResultSize: queryRestriction.MaximumResultSize,
		}
		if request.Schema != nil {
			queryContext.Schema = *request.Schema
		}
		result, duration, queryErr := executeWithTimeout(ctx, s.store, s.licenseService, driver, conn, request.Statement, queryContext)

		s.createQueryHistory(database, store.QueryHistoryTypeQuery, request.Statement, user.ID, duration, queryErr)
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

		if err := stream.Send(response); err != nil {
			return connect.NewError(connect.CodeInternal, errors.Errorf("failed to send response: %v", err))
		}
	}
}

func (s *SQLService) Query(ctx context.Context, req *connect.Request[v1pb.QueryRequest]) (*connect.Response[v1pb.QueryResponse], error) {
	request := req.Msg
	// Prepare related message.
	user, instance, database, err := s.prepareRelatedMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}

	statement := request.Statement
	// In Redshift datashare, Rewrite query used for parser.
	if database.Metadata.GetDatashare() {
		statement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", database.DatabaseName), "")
	}

	// Validate the request.
	// New query ACL experience.
	if !request.Explain && !common.EngineSupportQueryNewACL(instance.Metadata.GetEngine()) {
		if err := validateQueryRequest(instance, statement); err != nil {
			return nil, err
		}
	}

	dataSource, err := checkAndGetDataSourceQueriable(ctx, s.store, s.licenseService, database, request.DataSourceId)
	if err != nil {
		return nil, err
	}
	driver, err := s.dbFactory.GetDataSourceDriver(ctx, instance, dataSource, db.ConnectionContext{
		DatabaseName: database.DatabaseName,
		DataShare:    database.Metadata.GetDatashare(),
		ReadOnly:     dataSource.GetType() == storepb.DataSourceType_READ_ONLY,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get database driver: %v", err))
	}
	defer driver.Close(ctx)

	sqlDB := driver.GetDB()
	var conn *sql.Conn
	if sqlDB != nil {
		conn, err = sqlDB.Conn(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get database connection: %v", err))
		}
		defer conn.Close()
	}

	startTime := time.Now()
	queryRestriction := getMaximumSQLResultLimit(ctx, s.store, s.licenseService, request.Limit)
	queryContext := db.QueryContext{
		Explain:              request.Explain,
		Limit:                int(queryRestriction.MaximumResultRows),
		OperatorEmail:        user.Email,
		Option:               request.QueryOption,
		Container:            request.GetContainer(),
		MaximumSQLResultSize: queryRestriction.MaximumResultSize,
	}
	if request.Schema != nil {
		queryContext.Schema = *request.Schema
	}
	results, _, duration, queryErr := queryRetryStopOnError(
		ctx,
		s.store,
		user,
		instance,
		database,
		driver,
		conn,
		statement,
		queryContext,
		s.licenseService,
		s.accessCheck,
		s.schemaSyncer,
		storepb.MaskingExceptionPolicy_MaskingException_QUERY,
	)
	slog.Debug("query finished",
		log.BBError(queryErr),
		slog.Duration("duration", time.Since(startTime)),
		slog.String("instance", instance.ResourceID),
		slog.String("database", database.DatabaseName),
	)

	// Update activity.
	s.createQueryHistory(database, store.QueryHistoryTypeQuery, statement, user.ID, duration, queryErr)

	if queryErr != nil {
		if len(results) == 0 {
			if _, ok := queryErr.(*connect.Error); ok {
				return nil, queryErr
			}
			return nil, connect.NewError(connect.CodeInternal, errors.New(queryErr.Error()))
		}
		// populate the detailed_error field of the last query result
		var qe *queryError
		var pe *parserbase.SyntaxError
		if errors.As(queryErr, &qe) {
			if len(qe.resources) > 0 {
				results[len(results)-1].DetailedError = &v1pb.QueryResult_PermissionDenied_{
					PermissionDenied: &v1pb.QueryResult_PermissionDenied{
						Resources: qe.resources,
					},
				}
			} else if qe.commandType != v1pb.QueryResult_PermissionDenied_COMMAND_TYPE_UNSPECIFIED {
				results[len(results)-1].DetailedError = &v1pb.QueryResult_PermissionDenied_{
					PermissionDenied: &v1pb.QueryResult_PermissionDenied{
						CommandType: qe.commandType,
					},
				}
			}
		} else if errors.As(queryErr, &pe) {
			results[len(results)-1].DetailedError = &v1pb.QueryResult_SyntaxError_{
				SyntaxError: &v1pb.QueryResult_SyntaxError{
					StartPosition: convertToPosition(pe.Position),
				},
			}
		}
	}

	slog.Debug("request finished",
		slog.Duration("duration", time.Since(startTime)),
		slog.String("instance", instance.ResourceID),
		slog.String("database", database.DatabaseName),
	)

	response := &v1pb.QueryResponse{
		Results: results,
	}

	return connect.NewResponse(response), nil
}

func getMaximumSQLResultLimit(
	ctx context.Context,
	stores *store.Store,
	licenseService *enterprise.LicenseService,
	limit int32,
) *storepb.QueryDataPolicy {
	value := &storepb.QueryDataPolicy{
		MaximumResultSize: common.DefaultMaximumSQLResultSize,
		MaximumResultRows: -1,
	}
	if err := licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_QUERY_POLICY); err == nil {
		policy, err := stores.GetQueryDataPolicy(ctx)
		if err != nil {
			slog.Error("failed to get the query data policy", log.BBError(err))
			return value
		}
		value = policy
	}
	if limit > 0 && (value.GetMaximumResultRows() < 0 || limit < value.GetMaximumResultRows()) {
		value.MaximumResultRows = limit
	}
	return value
}

type accessCheckFunc func(context.Context, *store.InstanceMessage, *store.DatabaseMessage, *store.UserMessage, []*parserbase.QuerySpan, bool /* isExplain */) error

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

func getSchemaMetadata(engine storepb.Engine, dbSchema *model.DatabaseSchema) *model.SchemaMetadata {
	switch engine {
	case storepb.Engine_POSTGRES:
		return dbSchema.GetDatabaseMetadata().GetSchema(common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES))
	case storepb.Engine_MSSQL:
		return dbSchema.GetDatabaseMetadata().GetSchema("dbo")
	default:
		return dbSchema.GetDatabaseMetadata().GetSchema("")
	}
}

func replaceBackupTableWithSource(ctx context.Context, stores *store.Store, instance *store.InstanceMessage, database *store.DatabaseMessage, spans []*parserbase.QuerySpan) error {
	switch instance.Metadata.GetEngine() {
	case storepb.Engine_POSTGRES:
		// Don't need to check the database name for postgres here.
		// We backup the table to the same database with bbdataarchive schema for Postgres.
	case storepb.Engine_ORACLE:
		if database.DatabaseName != common.BackupDatabaseNameOfEngine(storepb.Engine_ORACLE) {
			return nil
		}
	default:
		if database.DatabaseName != common.BackupDatabaseNameOfEngine(instance.Metadata.GetEngine()) {
			return nil
		}
	}
	dbSchema, err := stores.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return err
	}
	schema := getSchemaMetadata(instance.Metadata.GetEngine(), dbSchema)
	if schema == nil {
		return nil
	}

	for _, span := range spans {
		span.SourceColumns = generateNewSourceColumnSet(instance.Metadata.GetEngine(), span.SourceColumns, schema)
		for _, result := range span.Results {
			result.SourceColumns = generateNewSourceColumnSet(instance.Metadata.GetEngine(), result.SourceColumns, schema)
		}
	}
	return nil
}

func generateNewSourceColumnSet(engine storepb.Engine, origin parserbase.SourceColumnSet, schema *model.SchemaMetadata) parserbase.SourceColumnSet {
	result := make(parserbase.SourceColumnSet)
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

func generateNewColumn(engine storepb.Engine, column parserbase.ColumnResource, database, schema, table string) parserbase.ColumnResource {
	switch engine {
	case storepb.Engine_POSTGRES:
		return parserbase.ColumnResource{
			Server:   column.Server,
			Database: column.Database,
			Schema:   database,
			Table:    table,
			Column:   column.Column,
		}
	default:
		return parserbase.ColumnResource{
			Server:   column.Server,
			Database: database,
			Schema:   schema,
			Table:    table,
			Column:   column.Column,
		}
	}
}

func isBackupTable(engine storepb.Engine, column parserbase.ColumnResource) bool {
	switch engine {
	case storepb.Engine_POSTGRES:
		return column.Schema == common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)
	case storepb.Engine_ORACLE:
		return column.Database == common.BackupDatabaseNameOfEngine(storepb.Engine_ORACLE)
	default:
		return column.Database == common.BackupDatabaseNameOfEngine(engine)
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
	queryContext db.QueryContext,
	licenseService *enterprise.LicenseService,
	optionalAccessCheck accessCheckFunc,
	schemaSyncer *schemasync.Syncer,
	action storepb.MaskingExceptionPolicy_MaskingException_Action,
) ([]*v1pb.QueryResult, []*parserbase.QuerySpan, time.Duration, error) {
	var spans []*parserbase.QuerySpan
	var sensitivePredicateColumns [][]parserbase.ColumnResource
	var err error
	if !queryContext.Explain {
		spans, err = parserbase.GetQuerySpan(
			ctx,
			parserbase.GetQuerySpanContext{
				InstanceID:                    instance.ResourceID,
				GetDatabaseMetadataFunc:       BuildGetDatabaseMetadataFunc(stores),
				ListDatabaseNamesFunc:         BuildListDatabaseNamesFunc(stores),
				GetLinkedDatabaseMetadataFunc: BuildGetLinkedDatabaseMetadataFunc(stores, instance.Metadata.GetEngine()),
			},
			instance.Metadata.GetEngine(),
			statement,
			database.DatabaseName,
			queryContext.Schema,
			!store.IsObjectCaseSensitive(instance),
		)
		if err != nil {
			return nil, nil, time.Duration(0), err
		}
		// After replacing backup table with source, we can apply the original access check and mask sensitive data for backup table.
		// If err != nil, this function will return the original spans.
		if err := replaceBackupTableWithSource(ctx, stores, instance, database, spans); err != nil {
			slog.Debug("failed to replace backup table with source", log.BBError(err))
		}
		if optionalAccessCheck != nil {
			// Check query access
			if err := optionalAccessCheck(ctx, instance, database, user, spans, queryContext.Explain); err != nil {
				return nil, nil, time.Duration(0), err
			}
			slog.Debug("optional access check", slog.String("instance", instance.ResourceID), slog.String("database", database.DatabaseName))
		}
		if licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_DATA_MASKING, instance) == nil {
			masker := NewQueryResultMasker(stores)
			sensitivePredicateColumns, err = masker.ExtractSensitivePredicateColumns(ctx, spans, instance, user, action)
			if err != nil {
				return nil, nil, time.Duration(0), connect.NewError(connect.CodeInternal, errors.New(err.Error()))
			}
			slog.Debug("extract sensitive predicate columns", slog.String("instance", instance.ResourceID), slog.String("database", database.DatabaseName))
		}
	}

	slog.Debug("start execute with timeout", slog.String("instance", instance.ResourceID), slog.String("database", database.DatabaseName), slog.String("statement", statement))
	results, duration, queryErr := executeWithTimeout(ctx, stores, licenseService, driver, conn, statement, queryContext)
	if queryErr != nil {
		return nil, nil, duration, queryErr
	}
	slog.Debug("execute success", slog.String("instance", instance.ResourceID), slog.String("statement", statement), slog.Duration("duration", duration))
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
				slog.Debug("database metadata need to sync", slog.String("instance", instance.ResourceID), slog.String("database", k.Database), slog.String("schema", k.Schema), slog.String("table", k.Table), slog.String("column", k.Column))
				syncDatabaseMap[k.Database] = true
			}
		}
	}

	// Sync database metadata.
	for accessDatabaseName := range syncDatabaseMap {
		slog.Debug("sync database metadata", slog.String("instance", instance.ResourceID), slog.String("database", accessDatabaseName))
		d, err := stores.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &accessDatabaseName})
		if err != nil {
			return nil, nil, duration, err
		}
		if err := schemaSyncer.SyncDatabaseSchema(ctx, d); err != nil {
			return nil, nil, duration, errors.Wrapf(err, "failed to sync database schema for database %q", accessDatabaseName)
		}
	}

	// Retry getting query span.
	if len(syncDatabaseMap) > 0 {
		slog.Debug("retry query after sync metadata", slog.String("instance", instance.ResourceID), slog.String("database", database.DatabaseName))
		spans, err = parserbase.GetQuerySpan(
			ctx,
			parserbase.GetQuerySpanContext{
				InstanceID:                    instance.ResourceID,
				GetDatabaseMetadataFunc:       BuildGetDatabaseMetadataFunc(stores),
				ListDatabaseNamesFunc:         BuildListDatabaseNamesFunc(stores),
				GetLinkedDatabaseMetadataFunc: BuildGetLinkedDatabaseMetadataFunc(stores, instance.Metadata.GetEngine()),
			},
			instance.Metadata.GetEngine(),
			statement,
			database.DatabaseName,
			queryContext.Schema,
			!store.IsObjectCaseSensitive(instance),
		)
		if err != nil {
			return nil, nil, time.Duration(0), err
		}
		// After replacing backup table with source, we can apply the original access check and mask sensitive data for backup table.
		// If err != nil, this function will return the original spans.
		if err := replaceBackupTableWithSource(ctx, stores, instance, database, spans); err != nil {
			slog.Debug("failed to replace backup table with source", log.BBError(err))
		}
		if licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_DATA_MASKING, instance) == nil {
			masker := NewQueryResultMasker(stores)
			sensitivePredicateColumns, err = masker.ExtractSensitivePredicateColumns(ctx, spans, instance, user, action)
			if err != nil {
				return nil, nil, time.Duration(0), connect.NewError(connect.CodeInternal, errors.New(err.Error()))
			}
		}
	}
	// The second query span should not tolerate any error, but we should retail the original error from database if possible.
	for i, result := range results {
		if i < len(spans) && result.Error == "" {
			if spans[i].FunctionNotSupportedError != nil {
				return nil, nil, duration, connect.NewError(connect.CodeInternal, errors.Errorf("failed to mask data: %v", spans[i].FunctionNotSupportedError))
			}
			if spans[i].NotFoundError != nil {
				return nil, nil, duration, connect.NewError(connect.CodeInternal, errors.Errorf("failed to mask data: %v", spans[i].NotFoundError))
			}
		}
	}

	if licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_DATA_MASKING, instance) == nil && !queryContext.Explain {
		slog.Debug("mask query results", slog.String("instance", instance.ResourceID), slog.String("database", database.DatabaseName))
		// TODO(zp): Refactor Document Database and RDBMS to use the same masking logic.
		if instance.Metadata.GetEngine() == storepb.Engine_COSMOSDB {
			if len(spans) != 1 {
				return nil, nil, duration, connect.NewError(connect.CodeInternal, errors.New("expected one span for CosmosDB"))
			}
			objectSchema, err := getCosmosDBContainerObjectSchema(ctx, stores, database.InstanceID, database.DatabaseName, queryContext.Container)
			if err != nil {
				return nil, nil, duration, connect.NewError(connect.CodeInternal, errors.New(err.Error()))
			}
			for pathStr, predicatePath := range spans[0].PredicatePaths {
				semanticType := getFirstSemanticTypeInPath(predicatePath, objectSchema)
				if semanticType != "" {
					for _, result := range results {
						result.Error = fmt.Sprintf("using path %q tagged by semantic type %q in WHERE clause is not allowed", pathStr, semanticType)
						result.Rows = nil
						result.RowsCount = 0
					}
					return results, spans, duration, nil
				}
			}
			if objectSchema != nil {
				// We store one query result document in one row.
				for _, result := range results {
					for _, row := range result.Rows {
						if len(row.Values) != 1 {
							continue
						}
						value := row.Values[0].GetStringValue()
						if value == "" {
							continue
						}
						semanticTypeToMaskerMap, err := buildSemanticTypeToMaskerMap(ctx, stores)
						if err != nil {
							return nil, nil, duration, connect.NewError(connect.CodeInternal, errors.New(err.Error()))
						}
						// Unmarshal the document.
						doc := make(map[string]any)
						if err := json.Unmarshal([]byte(value), &doc); err != nil {
							return nil, nil, duration, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal document: %v", err))
						}
						// Mask the document.
						maskedDoc, err := maskCosmosDB(spans[0], doc, objectSchema, semanticTypeToMaskerMap)
						if err != nil {
							return nil, nil, duration, connect.NewError(connect.CodeInternal, errors.Errorf("failed to mask document: %v", err))
						}
						// Marshal the masked document.
						maskedValue, err := json.Marshal(maskedDoc)
						if err != nil {
							return nil, nil, duration, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal masked document: %v", err))
						}
						row.Values[0] = &v1pb.RowValue{
							Kind: &v1pb.RowValue_StringValue{
								StringValue: string(maskedValue),
							},
						}
					}
				}
			}
		} else {
			masker := NewQueryResultMasker(stores)
			if err := masker.MaskResults(ctx, spans, results, instance, user, action); err != nil {
				return nil, nil, duration, connect.NewError(connect.CodeInternal, errors.New(err.Error()))
			}

			for i, result := range results {
				if i >= len(sensitivePredicateColumns) {
					continue
				}
				if len(sensitivePredicateColumns[i]) == 0 {
					continue
				}
				result.Error = getSensitivePredicateColumnErrorMessages(sensitivePredicateColumns[i])
				result.Rows = nil
				result.RowsCount = 0
			}
		}
	}
	return results, spans, duration, nil
}

func getCosmosDBContainerObjectSchema(ctx context.Context, stores *store.Store, instanceID string, databaseName string, containerName string) (*storepb.ObjectSchema, error) {
	dbSchema, err := stores.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database schema: %q", databaseName)
	}

	if dbSchema == nil {
		return nil, nil
	}

	schemas := dbSchema.GetConfig().GetSchemas()
	if len(schemas) == 0 {
		return nil, nil
	}

	schema := schemas[0]
	tables := schema.GetTables()
	for _, table := range tables {
		if table.GetName() == containerName {
			return table.GetObjectSchema(), nil
		}
	}

	return nil, nil
}

func getSensitivePredicateColumnErrorMessages(sensitiveColumns []parserbase.ColumnResource) string {
	var buf bytes.Buffer
	_, _ = buf.WriteString("Using sensitive columns in WHERE clause is not allowed: ")
	for j, column := range sensitiveColumns {
		if j > 0 {
			_, _ = buf.WriteString(", ")
		}
		_, _ = buf.WriteString(column.String())
	}
	return buf.String()
}

// queryRetryStopOnError runs the query and stops on encountering errors.
// The error is both present in the returned QueryResult and error, the caller decides what to do.
func queryRetryStopOnError(
	ctx context.Context,
	stores *store.Store,
	user *store.UserMessage,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	driver db.Driver,
	conn *sql.Conn,
	statement string,
	queryContext db.QueryContext,
	licenseService *enterprise.LicenseService,
	optionalAccessCheck accessCheckFunc,
	schemaSyncer *schemasync.Syncer,
	action storepb.MaskingExceptionPolicy_MaskingException_Action,
) ([]*v1pb.QueryResult, []*parserbase.QuerySpan, time.Duration, error) {
	// Split the statement into individual SQLs
	statements, err := parserbase.SplitMultiSQL(instance.Metadata.GetEngine(), statement)
	if err != nil {
		// Fall back to executing as a single statement if splitting fails
		return queryRetry(ctx, stores, user, instance, database, driver, conn, statement, queryContext, licenseService, optionalAccessCheck, schemaSyncer, action)
	}

	var allResults []*v1pb.QueryResult
	var allSpans []*parserbase.QuerySpan
	var totalDuration time.Duration

	for _, stmt := range statements {
		// Skip empty statements
		if stmt.Empty {
			continue
		}

		results, spans, duration, err := queryRetry(ctx, stores, user, instance, database, driver, conn, stmt.Text, queryContext, licenseService, optionalAccessCheck, schemaSyncer, action)
		totalDuration += duration

		if err != nil {
			allResults = append(allResults, &v1pb.QueryResult{
				Error:     err.Error(),
				Statement: stmt.Text,
			})
			allSpans = append(allSpans, nil)
			return allResults, allSpans, totalDuration, err
		}

		allResults = append(allResults, results...)
		allSpans = append(allSpans, spans...)

		// results may have swollen error.
		for _, result := range results {
			if result.Error != "" {
				return allResults, allSpans, totalDuration, nil
			}
		}
	}

	return allResults, allSpans, totalDuration, nil
}

func executeWithTimeout(ctx context.Context, stores *store.Store, licenseService *enterprise.LicenseService, driver db.Driver, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, time.Duration, error) {
	queryCtx := ctx
	var timeout time.Duration
	// For access control feature, we will use the timeout from request and query data policy.
	// Otherwise, no timeout will be applied.
	if licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_QUERY_POLICY) == nil {
		queryDataPolicy, err := stores.GetQueryDataPolicy(ctx)
		if err != nil {
			return nil, time.Duration(0), errors.Wrap(err, "failed to get query data policy")
		}
		// Override the timeout if the query data policy has a smaller timeout.
		if queryDataPolicy.Timeout.GetSeconds() > 0 || queryDataPolicy.Timeout.GetNanos() > 0 {
			timeout = queryDataPolicy.Timeout.AsDuration()
			slog.Debug("create query context with timeout", slog.Duration("timeout", timeout))
			newCtx, cancelCtx := context.WithTimeout(ctx, timeout)
			defer cancelCtx()
			queryCtx = newCtx
		}
	}
	start := time.Now()
	result, err := driver.QueryConn(queryCtx, conn, statement, queryContext)
	select {
	case <-queryCtx.Done():
		// canceled or timed out
		return nil, time.Since(start), errors.Errorf("timeout reached: %v", timeout)
	default:
		// So the select will not block
	}
	sanitizeResults(result)
	return result, time.Since(start), err
}

// Export exports the SQL query result.
func (s *SQLService) Export(ctx context.Context, req *connect.Request[v1pb.ExportRequest]) (*connect.Response[v1pb.ExportResponse], error) {
	request := req.Msg
	// Prehandle export from issue.
	if strings.HasPrefix(request.Name, common.ProjectNamePrefix) {
		response, err := s.doExportFromIssue(ctx, request.Name)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(response), nil
	}

	// Check if data export is allowed.
	queryDataPolicy, err := s.store.GetQueryDataPolicy(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get data export policy: %v", err))
	}
	if queryDataPolicy.DisableExport {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("data export is not allowed"))
	}

	// Prepare related message.
	user, instance, database, err := s.prepareRelatedMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}

	statement := request.Statement
	// In Redshift datashare, Rewrite query used for parser.
	if database.Metadata.GetDatashare() {
		statement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", database.DatabaseName), "")
	}

	// Validate the request.
	// New query ACL experience.
	if instance.Metadata.GetEngine() != storepb.Engine_MYSQL {
		if err := validateQueryRequest(instance, statement); err != nil {
			return nil, err
		}
	}

	dataSource, err := checkAndGetDataSourceQueriable(ctx, s.store, s.licenseService, database, request.DataSourceId)
	if err != nil {
		return nil, err
	}
	bytes, duration, exportErr := DoExport(ctx, s.store, s.dbFactory, s.licenseService, request, user, instance, database, s.accessCheck, s.schemaSyncer, dataSource)

	s.createQueryHistory(database, store.QueryHistoryTypeExport, statement, user.ID, duration, exportErr)

	if exportErr != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New(exportErr.Error()))
	}

	return connect.NewResponse(&v1pb.ExportResponse{
		Content: bytes,
	}), nil
}

func (s *SQLService) doExportFromIssue(ctx context.Context, requestName string) (*v1pb.ExportResponse, error) {
	// Try to parse as rollout name first (more specific), then fallback to stage name
	var rolloutID int
	var projectID string
	var err error
	projectID, rolloutID, err = common.GetProjectIDRolloutID(requestName)
	if err != nil {
		// If rollout parsing fails, try parsing as stage name
		projectID, rolloutID, _, err = common.GetProjectIDRolloutIDMaybeStageID(requestName)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse request name as rollout or stage: %v", err))
		}
	}

	pipeline, err := s.store.GetPipelineV2(ctx, &store.PipelineFind{
		ID:        &rolloutID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get rollout: %v", err))
	}
	if pipeline == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout %d not found in project %s", rolloutID, projectID))
	}

	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: &pipeline.ID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get tasks: %v", err))
	}
	if len(tasks) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("rollout %d has no task", pipeline.ID))
	}

	pendingEncrypts := []*encryptContent{}
	targetTaskRunStatus := []storepb.TaskRun_Status{storepb.TaskRun_DONE}

	for _, task := range tasks {
		taskRuns, err := s.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{
			TaskUID: &task.ID,
			Status:  &targetTaskRunStatus,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get task run: %v", err))
		}
		if len(taskRuns) == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("rollout %v has no task run", requestName))
		}
		taskRun := taskRuns[0]
		exportArchiveUID := int(taskRun.ResultProto.ExportArchiveUid)
		if exportArchiveUID == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("issue %v has no export archive", requestName))
		}
		exportArchive, err := s.store.GetExportArchive(ctx, &store.FindExportArchiveMessage{UID: &exportArchiveUID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get export archive: %v", err))
		}
		if exportArchive == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("export not found or expired, please request a new export"))
		}

		// The exportArchive.Bytes should be a zip without password. We will read it and append all files into the pendingEncrypts,
		// then create a new file zip for them.
		zipReader, err := zip.NewReader(bytes.NewReader(exportArchive.Bytes), int64(len(exportArchive.Bytes)))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to read export archive: %v", err))
		}

		for _, file := range zipReader.File {
			rc, err := file.Open()
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to open file %s in archive: %v", file.Name, err))
			}

			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to read file %s: %v", file.Name, err))
			}

			pendingEncrypts = append(pendingEncrypts, &encryptContent{
				Content: content,
				Name:    file.Name,
			})
		}
	}

	encryptedBytes, err := doEncrypt(pendingEncrypts, tasks[0].Payload.GetPassword())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to encrypt data: %v", err))
	}

	return &v1pb.ExportResponse{
		Content: encryptedBytes,
	}, nil
}

// DoExport does the export.
func DoExport(
	ctx context.Context,
	stores *store.Store,
	dbFactory *dbfactory.DBFactory,
	licenseService *enterprise.LicenseService,
	request *v1pb.ExportRequest,
	user *store.UserMessage,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	optionalAccessCheck accessCheckFunc,
	schemaSyncer *schemasync.Syncer,
	dataSource *storepb.DataSource,
) ([]byte, time.Duration, error) {
	if dataSource == nil {
		return nil, 0, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot found valid data source"))
	}
	driver, err := dbFactory.GetDataSourceDriver(ctx, instance, dataSource, db.ConnectionContext{
		DatabaseName: database.DatabaseName,
		DataShare:    database.Metadata.GetDatashare(),
		ReadOnly:     true,
	})
	if err != nil {
		return nil, 0, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get database driver: %v", err))
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
	queryRestriction := getMaximumSQLResultLimit(ctx, stores, licenseService, request.Limit)
	queryContext := db.QueryContext{
		Limit:                int(queryRestriction.MaximumResultRows),
		OperatorEmail:        user.Email,
		MaximumSQLResultSize: queryRestriction.MaximumResultSize,
	}
	if request.Schema != nil {
		queryContext.Schema = *request.Schema
	}
	results, spans, duration, queryErr := queryRetry(
		ctx,
		stores,
		user,
		instance,
		database,
		driver,
		conn,
		request.Statement,
		queryContext,
		licenseService,
		optionalAccessCheck,
		schemaSyncer,
		storepb.MaskingExceptionPolicy_MaskingException_EXPORT,
	)
	if queryErr != nil {
		return nil, duration, queryErr
	}

	if licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_DATA_MASKING, instance) == nil {
		masker := NewQueryResultMasker(stores)
		if err := masker.MaskResults(ctx, spans, results, instance, user, storepb.MaskingExceptionPolicy_MaskingException_EXPORT); err != nil {
			return nil, duration, err
		}
	}

	var buf bytes.Buffer
	zipw := zip.NewWriter(&buf)

	exportCount := 0
	for i, result := range results {
		if result.GetError() != "" {
			logExportError(database, "failed to query result", errors.New(result.GetError()))
			continue
		}

		if err := exportResultToZip(ctx, zipw, stores, instance, database, result, request, i+1); err != nil {
			logExportError(database, "failed to export result to zip", err)
			continue
		}

		exportCount++
		// Help GC by clearing the result data we've already processed
		result.Rows = nil
	}

	if exportCount == 0 {
		return nil, duration, errors.Errorf("empty export data for database %s", database.DatabaseName)
	}

	if err := zipw.Close(); err != nil {
		return nil, duration, errors.Wrap(err, "failed to close zip writer")
	}

	return buf.Bytes(), duration, nil
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

// exportResultToZip exports a single query result to the ZIP archive.
// It writes both the SQL statement and the formatted result data.
func exportResultToZip(
	ctx context.Context,
	zipw *zip.Writer,
	stores *store.Store,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	result *v1pb.QueryResult,
	request *v1pb.ExportRequest,
	statementNumber int,
) error {
	baseFilename := fmt.Sprintf("%s/%s/statement-%d", database.InstanceID, database.DatabaseName, statementNumber)

	// Write statement file
	statementFilename := fmt.Sprintf("%s.sql", baseFilename)
	if err := export.WriteZipEntry(zipw, statementFilename, []byte(result.Statement), request.GetPassword()); err != nil {
		return errors.Wrap(err, "failed to write statement")
	}

	// Write result file by streaming directly to ZIP
	resultExt := strings.ToLower(request.Format.String())
	resultFilename := fmt.Sprintf("%s.result.%s", baseFilename, resultExt)
	if err := formatExportToZip(ctx, zipw, resultFilename, stores, instance, database, result, request); err != nil {
		return errors.Wrap(err, "failed to write formatted result")
	}

	return nil
}

// formatExportToZip formats query results and writes them directly to a ZIP entry.
// This function streams the formatted data to minimize memory usage.
func formatExportToZip(
	ctx context.Context,
	zipw *zip.Writer,
	filename string,
	stores *store.Store,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	result *v1pb.QueryResult,
	request *v1pb.ExportRequest,
) error {
	writer, err := export.CreateZipWriter(zipw, filename, request.GetPassword())
	if err != nil {
		return err
	}

	return writeFormattedResult(ctx, writer, stores, instance, database, result, request)
}

// writeFormattedResult writes the query result in the requested format to the writer.
func writeFormattedResult(
	ctx context.Context,
	w io.Writer,
	stores *store.Store,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	result *v1pb.QueryResult,
	request *v1pb.ExportRequest,
) error {
	switch request.Format {
	case v1pb.ExportFormat_CSV:
		return export.CSVToWriter(w, result)
	case v1pb.ExportFormat_JSON:
		return export.JSONToWriter(w, result)
	case v1pb.ExportFormat_SQL:
		return exportSQLWithContext(ctx, w, stores, instance, database, result, request)
	case v1pb.ExportFormat_XLSX:
		return export.XLSXToWriter(w, result)
	default:
		return errors.Errorf("unsupported export format: %s", request.Format.String())
	}
}

// exportSQLWithContext exports SQL INSERT statements with proper context.
func exportSQLWithContext(
	ctx context.Context,
	w io.Writer,
	stores *store.Store,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	result *v1pb.QueryResult,
	request *v1pb.ExportRequest,
) error {
	resourceList, err := export.GetResources(
		ctx,
		stores,
		instance.Metadata.GetEngine(),
		database.DatabaseName,
		request.Statement,
		instance,
		BuildGetDatabaseMetadataFunc(stores),
		BuildListDatabaseNamesFunc(stores),
		BuildGetLinkedDatabaseMetadataFunc(stores, instance.Metadata.GetEngine()),
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

type encryptContent struct {
	Name    string
	Content []byte
}

func doEncrypt(exports []*encryptContent, password string) ([]byte, error) {
	var b bytes.Buffer
	fzip := io.Writer(&b)

	zipw := zip.NewWriter(fzip)
	defer zipw.Close()

	for _, exportContent := range exports {
		if err := export.WriteZipEntry(zipw, exportContent.Name, exportContent.Content, password); err != nil {
			return nil, err
		}
	}

	if err := zipw.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close zip writer")
	}

	return b.Bytes(), nil
}

func (s *SQLService) createQueryHistory(database *store.DatabaseMessage, queryType store.QueryHistoryType, statement string, userUID int, duration time.Duration, queryErr error) {
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

	// Use a fresh context with timeout for creating query history
	// to avoid being affected by request cancellation
	historyCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := s.store.CreateQueryHistory(historyCtx, qh); err != nil {
		queryErr := ""
		if v := qh.Payload.Error; v != nil {
			queryErr = *v
		}
		slog.Error(
			"failed to create query history",
			log.BBError(err),
			slog.String("instance", database.InstanceID),
			slog.String("database", database.DatabaseName),
			slog.String("project", database.ProjectID),
			slog.String("query_error", queryErr),
		)
	}
}

// SearchQueryHistories lists query histories.
func (s *SQLService) SearchQueryHistories(ctx context.Context, req *connect.Request[v1pb.SearchQueryHistoriesRequest]) (*connect.Response[v1pb.SearchQueryHistoriesResponse], error) {
	request := req.Msg
	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.PageToken,
		limit:   int(request.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	find := &store.FindQueryHistoryMessage{
		CreatorUID: &user.ID,
		Limit:      &limitPlusOne,
		Offset:     &offset.offset,
	}
	filterQ, err := store.GetListQueryHistoryFilter(request.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.FilterQ = filterQ

	historyList, err := s.store.ListQueryHistories(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list history: %v", err.Error()))
	}

	nextPageToken := ""
	if len(historyList) == limitPlusOne {
		historyList = historyList[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal next page token, error: %v", err))
		}
	}

	resp := &v1pb.SearchQueryHistoriesResponse{
		NextPageToken: nextPageToken,
	}
	for _, history := range historyList {
		queryHistory, err := s.convertToV1QueryHistory(ctx, history)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert log entity, error: %v", err))
		}
		if queryHistory == nil {
			continue
		}
		resp.QueryHistories = append(resp.QueryHistories, queryHistory)
	}

	return connect.NewResponse(resp), nil
}

func BuildGetLinkedDatabaseMetadataFunc(storeInstance *store.Store, engine storepb.Engine) parserbase.GetLinkedDatabaseMetadataFunc {
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
			meta, err := storeInstance.GetDBSchema(ctx, &store.FindDBSchemaMessage{
				InstanceID:   database.InstanceID,
				DatabaseName: database.DatabaseName,
			})
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
			instance, err := storeInstance.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
			if err != nil {
				return "", "", nil, err
			}
			if instance != nil {
				for _, dataSource := range instance.Metadata.DataSources {
					if strings.Contains(linkedMeta.GetHost(), dataSource.GetHost()) {
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
		linkedDatabaseMetadata, err := storeInstance.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   linkedDatabase.InstanceID,
			DatabaseName: linkedDatabase.DatabaseName,
		})
		if err != nil {
			return "", "", nil, err
		}
		if linkedDatabaseMetadata == nil {
			return "", "", nil, nil
		}
		return linkedDatabase.InstanceID, linkedDatabaseName, linkedDatabaseMetadata.GetDatabaseMetadata(), nil
	}
}

func BuildGetDatabaseMetadataFunc(storeInstance *store.Store) parserbase.GetDatabaseMetadataFunc {
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
		databaseMetadata, err := storeInstance.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   instanceID,
			DatabaseName: databaseName,
		})
		if err != nil {
			return "", nil, err
		}
		if databaseMetadata == nil {
			return "", nil, nil
		}
		return databaseName, databaseMetadata.GetDatabaseMetadata(), nil
	}
}

func BuildListDatabaseNamesFunc(storeInstance *store.Store) parserbase.ListDatabaseNamesFunc {
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
	spans []*parserbase.QuerySpan,
	isExplain bool,
) error {
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return err
	}
	if project == nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("project %q not found", database.ProjectID))
	}

	for _, span := range spans {
		// New query ACL experience.
		if common.EngineSupportQueryNewACL(instance.Metadata.GetEngine()) {
			var permission iam.Permission
			switch span.Type {
			case parserbase.QueryTypeUnknown:
				return connect.NewError(connect.CodePermissionDenied, errors.New("disallowed query type"))
			case parserbase.DDL:
				permission = iam.PermissionSQLDdl
			case parserbase.DML:
				permission = iam.PermissionSQLDml
			case parserbase.Explain:
				permission = iam.PermissionSQLExplain
			case parserbase.SelectInfoSchema:
				permission = iam.PermissionSQLInfo
			case parserbase.Select:
				// Conditional permission check below.
			default:
			}
			if isExplain {
				permission = iam.PermissionSQLExplain
			}
			if span.Type == parserbase.DDL || span.Type == parserbase.DML {
				if err := checkDataSourceQueryPolicy(ctx, s.store, s.licenseService, database, span.Type); err != nil {
					return err
				}
			}
			if permission != "" {
				ok, err := s.iamManager.CheckPermission(ctx, permission, user, project.ResourceID)
				if err != nil {
					return connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
				}
				if !ok {
					return connect.NewError(connect.CodePermissionDenied, errors.Errorf("user %q does not have permission %q on project %q", user.Email, permission, project.ResourceID))
				}
			}
		}
		if span.Type == parserbase.Select {
			var deniedResources []string
			for column := range span.SourceColumns {
				attributes := map[string]any{
					common.CELAttributeRequestTime:        time.Now(),
					common.CELAttributeResourceDatabase:   common.FormatDatabase(instance.ResourceID, column.Database),
					common.CELAttributeResourceSchemaName: column.Schema,
					common.CELAttributeResourceTableName:  column.Table,
				}

				databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
					InstanceID:      &instance.ResourceID,
					DatabaseName:    &column.Database,
					IsCaseSensitive: store.IsObjectCaseSensitive(instance),
				})
				if err != nil {
					return err
				}
				if databaseMessage == nil {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("database %q not found", column.Database))
				}
				project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &databaseMessage.ProjectID})
				if err != nil {
					return err
				}
				if project == nil {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("project %q not found", databaseMessage.ProjectID))
				}

				workspacePolicy, err := s.store.GetWorkspaceIamPolicy(ctx)
				if err != nil {
					return connect.NewError(connect.CodeInternal, errors.Errorf("failed to get workspace iam policy, error: %v", err))
				}
				// Allow query databases across different projects.
				projectPolicy, err := s.store.GetProjectIamPolicy(ctx, project.ResourceID)
				if err != nil {
					return connect.NewError(connect.CodeInternal, errors.New(err.Error()))
				}

				ok, err := s.hasDatabaseAccessRights(ctx, user, []*storepb.IamPolicy{workspacePolicy.Policy, projectPolicy.Policy}, attributes)
				if err != nil {
					return connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access control for database: %q, error %v", column.Database, err))
				}
				if !ok {
					resource, ok := attributes[common.CELAttributeResourceDatabase].(string)
					if !ok {
						resource = ""
					}
					if schema, ok := attributes[common.CELAttributeResourceSchemaName]; ok && schema != "" {
						resource = fmt.Sprintf("%s/schemas/%s", resource, schema)
					}
					if table, ok := attributes[common.CELAttributeResourceTableName]; ok && table != "" {
						resource = fmt.Sprintf("%s/tables/%s", resource, table)
					}
					deniedResources = append(deniedResources, resource)
				}
			}
			if len(deniedResources) > 0 {
				return &queryError{
					err: connect.NewError(
						connect.CodePermissionDenied,
						errors.Errorf("permission denied to access resources: %v", deniedResources),
					),
					resources: deniedResources,
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
		return nil, nil, nil, connect.NewError(connect.CodeInternal, errors.New(err.Error()))
	}

	database, err := getDatabaseMessage(ctx, s.store, requestName)
	if err != nil {
		return nil, nil, nil, connect.NewError(connect.CodeInternal, errors.New(err.Error()))
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &database.InstanceID,
	})
	if err != nil {
		return nil, nil, nil, connect.NewError(connect.CodeInternal, errors.New(err.Error()))
	}
	if instance == nil {
		return nil, nil, nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", database.InstanceID))
	}

	return user, instance, database, nil
}

func validateQueryRequest(instance *store.InstanceMessage, statement string) error {
	ok, _, err := parserbase.ValidateSQLForEditor(instance.Metadata.GetEngine(), statement)
	if err != nil {
		if syntaxErr, ok := err.(*parserbase.SyntaxError); ok {
			err := connect.NewError(connect.CodeInvalidArgument, syntaxErr)
			if detail, detailErr := connect.NewErrorDetail(&v1pb.PlanCheckRun_Result{
				Code:    int32(code.StatementSyntaxError),
				Content: syntaxErr.Message,
				Title:   "Syntax error",
				Status:  v1pb.Advice_ERROR,
				Report: &v1pb.PlanCheckRun_Result_SqlReviewReport_{
					SqlReviewReport: &v1pb.PlanCheckRun_Result_SqlReviewReport{
						StartPosition: convertToPosition(syntaxErr.Position),
					},
				},
			}); detailErr == nil {
				err.AddDetail(detail)
			}
			return err
		}
		return err
	}
	if !ok {
		return &queryError{
			err:         connect.NewError(connect.CodeInvalidArgument, errors.New("Support read-only command statements only")),
			commandType: v1pb.QueryResult_PermissionDenied_NON_READ_ONLY,
		}
	}
	return nil
}

func (s *SQLService) hasDatabaseAccessRights(ctx context.Context, user *store.UserMessage, iamPolicies []*storepb.IamPolicy, attributes map[string]any) (bool, error) {
	wantPermission := iam.PermissionSQLSelect

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
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("the user has been deactivated"))
	}
	return user, nil
}

func getClassificationByProject(ctx context.Context, stores *store.Store, projectID string) *storepb.DataClassificationSetting_DataClassificationConfig {
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

func (*SQLService) DiffMetadata(_ context.Context, req *connect.Request[v1pb.DiffMetadataRequest]) (*connect.Response[v1pb.DiffMetadataResponse], error) {
	request := req.Msg
	switch request.Engine {
	case v1pb.Engine_MYSQL, v1pb.Engine_POSTGRES, v1pb.Engine_TIDB, v1pb.Engine_ORACLE, v1pb.Engine_MSSQL:
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported engine: %v", request.Engine))
	}
	if request.SourceMetadata == nil || request.TargetMetadata == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("source_metadata and target_metadata are required"))
	}
	storeSourceMetadata := convertV1DatabaseMetadata(request.SourceMetadata)

	sourceConfig := convertDatabaseCatalog(request.GetSourceCatalog())
	sanitizeCommentForSchemaMetadata(storeSourceMetadata, model.NewDatabaseConfig(sourceConfig), request.ClassificationFromConfig)

	storeTargetMetadata := convertV1DatabaseMetadata(request.TargetMetadata)

	targetConfig := convertDatabaseCatalog(request.GetTargetCatalog())
	if err := checkDatabaseMetadata(storepb.Engine(request.Engine), storeTargetMetadata); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid target metadata"))
	}
	sanitizeCommentForSchemaMetadata(storeTargetMetadata, model.NewDatabaseConfig(targetConfig), request.ClassificationFromConfig)

	// Convert metadata to model.DatabaseSchema for diffing
	isObjectCaseSensitive := true
	sourceDBSchema := model.NewDatabaseSchema(storeSourceMetadata, nil, nil, storepb.Engine(request.Engine), isObjectCaseSensitive)
	targetDBSchema := model.NewDatabaseSchema(storeTargetMetadata, nil, nil, storepb.Engine(request.Engine), isObjectCaseSensitive)

	// Get the metadata diff between source and target
	metadataDiff, err := schema.GetDatabaseSchemaDiff(storepb.Engine(request.Engine), sourceDBSchema, targetDBSchema)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to compute diff between source and target schemas, error: %v", err))
	}

	// Generate migration SQL from the diff
	diff, err := schema.GenerateMigration(storepb.Engine(request.Engine), metadataDiff)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate migration SQL, error: %v", err))
	}

	return connect.NewResponse(&v1pb.DiffMetadataResponse{
		Diff: diff,
	}), nil
}

func sanitizeCommentForSchemaMetadata(dbSchema *storepb.DatabaseSchemaMetadata, dbModelConfig *model.DatabaseConfig, classificationFromConfig bool) {
	for _, schema := range dbSchema.Schemas {
		schemaConfig := dbModelConfig.GetSchemaConfig(schema.Name)
		for _, table := range schema.Tables {
			tableConfig := schemaConfig.GetTableConfig(table.Name)
			classificationID := ""
			if !classificationFromConfig {
				classificationID = tableConfig.Classification
			}
			table.Comment = common.GetCommentFromClassificationAndUserComment(classificationID, table.UserComment)
			for _, col := range table.Columns {
				columnConfig := tableConfig.GetColumnConfig(col.Name)
				classificationID := ""
				if !classificationFromConfig {
					classificationID = columnConfig.Classification
				}
				col.Comment = common.GetCommentFromClassificationAndUserComment(classificationID, col.UserComment)
			}
		}
	}
}

// GetQueriableDataSource try to returns the RO data source, and will returns the admin data source if not exist the RO data source.
func GetQueriableDataSource(instance *store.InstanceMessage) *storepb.DataSource {
	if len(instance.Metadata.GetDataSources()) == 0 {
		return nil
	}
	for _, ds := range instance.Metadata.GetDataSources() {
		if ds.GetType() == storepb.DataSourceType_READ_ONLY {
			return ds
		}
	}
	return instance.Metadata.DataSources[0]
}

func checkAndGetDataSourceQueriable(
	ctx context.Context,
	storeInstance *store.Store,
	licenseService *enterprise.LicenseService,
	database *store.DatabaseMessage,
	dataSourceID string,
) (*storepb.DataSource, error) {
	if dataSourceID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("data source id is required"))
	}

	instance, err := storeInstance.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get instance %v with error: %v", database.InstanceID, err.Error()))
	}
	if instance == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", database.InstanceID))
	}
	dataSource := func() *storepb.DataSource {
		for _, ds := range instance.Metadata.GetDataSources() {
			if ds.GetId() == dataSourceID {
				return ds
			}
		}
		return nil
	}()
	if dataSource == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("data source %q not found", dataSourceID))
	}

	// Always allow non-admin data source.
	if dataSource.GetType() != storepb.DataSourceType_ADMIN {
		if err := licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_INSTANCE_READ_ONLY_CONNECTION, instance); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New(err.Error()))
		}
		return dataSource, nil
	}

	//nolint:nilerr
	if err := licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_QUERY_POLICY); err != nil {
		return dataSource, nil
	}

	dataSourceQueryPolicyType := storepb.Policy_DATA_SOURCE_QUERY

	// get data source restriction policy for environment
	var envAdminDataSourceRestriction v1pb.DataSourceQueryPolicy_Restriction
	effectiveEnvironmentID := ""
	if database.EffectiveEnvironmentID != nil {
		effectiveEnvironmentID = *database.EffectiveEnvironmentID
	}
	if effectiveEnvironmentID != "" {
		environment, err := storeInstance.GetEnvironmentByID(ctx, effectiveEnvironmentID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get environment %s with error %v", effectiveEnvironmentID, err.Error()))
		}
		if environment == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("environment %q not found", effectiveEnvironmentID))
		}

		environmentResourceType := storepb.Policy_ENVIRONMENT
		environmentResource := common.FormatEnvironment(environment.Id)
		environmentPolicy, err := storeInstance.GetPolicyV2(ctx, &store.FindPolicyMessage{
			ResourceType: &environmentResourceType,
			Resource:     &environmentResource,
			Type:         &dataSourceQueryPolicyType,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get environment data source policy with error: %v", err.Error()))
		}
		if environmentPolicy != nil {
			envPayload, err := convertToV1PBDataSourceQueryPolicy(environmentPolicy.Payload)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert environment data source policy payload with error: %v", err.Error()))
			}
			envAdminDataSourceRestriction = envPayload.DataSourceQueryPolicy.GetAdminDataSourceRestriction()
		}
	}

	// get data source restriction policy for project
	var projectAdminDataSourceRestriction v1pb.DataSourceQueryPolicy_Restriction
	projectResourceType := storepb.Policy_PROJECT
	projectResource := common.FormatProject(database.ProjectID)
	projectPolicy, err := storeInstance.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &projectResourceType,
		Resource:     &projectResource,
		Type:         &dataSourceQueryPolicyType,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project data source policy with error: %v", err.Error()))
	}
	if projectPolicy != nil {
		projectPayload, err := convertToV1PBDataSourceQueryPolicy(projectPolicy.Payload)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert project data source policy payload with error: %v", err.Error()))
		}
		projectAdminDataSourceRestriction = projectPayload.DataSourceQueryPolicy.GetAdminDataSourceRestriction()
	}

	// If any of the policy is DISALLOW, then return false.
	if envAdminDataSourceRestriction == v1pb.DataSourceQueryPolicy_DISALLOW || projectAdminDataSourceRestriction == v1pb.DataSourceQueryPolicy_DISALLOW {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("data source %q is not queryable", dataSourceID))
	} else if envAdminDataSourceRestriction == v1pb.DataSourceQueryPolicy_FALLBACK || projectAdminDataSourceRestriction == v1pb.DataSourceQueryPolicy_FALLBACK {
		// If there is any read-only data source, then return false.
		if ds := GetQueriableDataSource(instance); ds != nil && ds.Type == storepb.DataSourceType_READ_ONLY {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("data source %q is not queryable", dataSourceID))
		}
	}

	return dataSource, nil
}

func checkDataSourceQueryPolicy(ctx context.Context, storeInstance *store.Store, licenseService *enterprise.LicenseService, database *store.DatabaseMessage, statementTp parserbase.QueryType) error {
	//nolint:nilerr
	if err := licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_QUERY_POLICY); err != nil {
		// If the feature is not enabled, then we don't need to check the policy.
		// For license backward compatibility.
		return nil
	}
	effectiveEnvironmentID := ""
	if database.EffectiveEnvironmentID != nil {
		effectiveEnvironmentID = *database.EffectiveEnvironmentID
	}
	if effectiveEnvironmentID == "" {
		return connect.NewError(connect.CodeNotFound, errors.New("no effective environment found for database"))
	}
	environment, err := storeInstance.GetEnvironmentByID(ctx, effectiveEnvironmentID)
	if err != nil {
		return err
	}
	if environment == nil {
		return connect.NewError(connect.CodeNotFound, errors.Errorf("environment %q not found", effectiveEnvironmentID))
	}
	resourceType := storepb.Policy_ENVIRONMENT
	environmentResource := common.FormatEnvironment(environment.Id)
	policyType := storepb.Policy_DATA_SOURCE_QUERY
	dataSourceQueryPolicy, err := storeInstance.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Resource:     &environmentResource,
		Type:         &policyType,
	})
	if err != nil {
		return err
	}
	if dataSourceQueryPolicy != nil {
		policy := &v1pb.DataSourceQueryPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(dataSourceQueryPolicy.Payload), policy); err != nil {
			return connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal data source query policy payload"))
		}
		switch statementTp {
		case parserbase.DDL:
			if policy.DisallowDdl {
				return &queryError{
					err:         connect.NewError(connect.CodePermissionDenied, errors.Errorf("disallow execute DDL statement in environment %q", environment.Title)),
					commandType: v1pb.QueryResult_PermissionDenied_DDL,
				}
			}
		case parserbase.DML:
			if policy.DisallowDml {
				return &queryError{
					err:         connect.NewError(connect.CodePermissionDenied, errors.Errorf("disallow execute DML statement in environment %q", environment.Title)),
					commandType: v1pb.QueryResult_PermissionDenied_DML,
				}
			}
		default:
		}
	}
	return nil
}
