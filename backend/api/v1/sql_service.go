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
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/transform"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
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
			for _, result := range response.Results {
				// The AdminExecute requires bb.sql.admin permission, so we can presume the users have enough permission to export.
				result.AllowExport = true
			}
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
	results, spans, duration, queryErr := queryRetry(
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

	// Update activity.
	if err = s.createQueryHistory(ctx, database, store.QueryHistoryTypeQuery, statement, user.ID, duration, queryErr); err != nil {
		return nil, err
	}
	if queryErr != nil {
		code := connect.CodeInternal
		// If queryErr is already a connect.Error, preserve its code
		if connectErr, ok := queryErr.(*connect.Error); ok {
			code = connectErr.Code()
		} else if syntaxErr, ok := queryErr.(*parserbase.SyntaxError); ok {
			err := connect.NewError(connect.CodeInvalidArgument, syntaxErr)
			if detail, detailErr := connect.NewErrorDetail(&v1pb.PlanCheckRun_Result{
				Code:    int32(advisor.StatementSyntaxError),
				Content: syntaxErr.Message,
				Title:   "Syntax error",
				Status:  v1pb.PlanCheckRun_Result_ERROR,
				Report: &v1pb.PlanCheckRun_Result_SqlReviewReport_{
					SqlReviewReport: &v1pb.PlanCheckRun_Result_SqlReviewReport{
						Line:   int32(syntaxErr.Position.GetLine()),
						Column: int32(syntaxErr.Position.GetColumn()),
					},
				},
			}); detailErr == nil {
				err.AddDetail(detail)
			}
			return nil, err
		}
		return nil, connect.NewError(code, errors.New(queryErr.Error()))
	}

	for _, result := range results {
		// AllowExport is a validate only check.
		checkErr := s.accessCheck(ctx, instance, database, user, spans, int(result.RowsCount), request.Explain, true /* isExport */)
		result.AllowExport = checkErr == nil
	}

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

type accessCheckFunc func(context.Context, *store.InstanceMessage, *store.DatabaseMessage, *store.UserMessage, []*parserbase.QuerySpan, int, bool /* isExplain */, bool /* isExport */) error

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
	dbSchema, err := stores.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
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
			if err := optionalAccessCheck(ctx, instance, database, user, spans, queryContext.Limit, queryContext.Explain, false); err != nil {
				return nil, nil, time.Duration(0), err
			}
		}
		if licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_DATA_MASKING, instance) == nil {
			masker := NewQueryResultMasker(stores)
			sensitivePredicateColumns, err = masker.ExtractSensitivePredicateColumns(ctx, spans, instance, user, action)
			if err != nil {
				return nil, nil, time.Duration(0), connect.NewError(connect.CodeInternal, errors.New(err.Error()))
			}
		}
	}

	results, duration, queryErr := executeWithTimeout(ctx, stores, licenseService, driver, conn, statement, queryContext)
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
		if err := schemaSyncer.SyncDatabaseSchema(ctx, d); err != nil {
			return nil, nil, duration, errors.Wrapf(err, "failed to sync database schema for database %q", accessDatabaseName)
		}
	}

	// Retry getting query span.
	if len(syncDatabaseMap) > 0 {
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
	dbSchema, err := stores.GetDBSchema(ctx, instanceID, databaseName)
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

	if err := s.createQueryHistory(ctx, database, store.QueryHistoryTypeExport, statement, user.ID, duration, exportErr); err != nil {
		return nil, err
	}

	if exportErr != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New(exportErr.Error()))
	}

	return connect.NewResponse(&v1pb.ExportResponse{
		Content: bytes,
	}), nil
}

func (s *SQLService) doExportFromIssue(ctx context.Context, requestName string) (*v1pb.ExportResponse, error) {
	_, rolloutID, _, err := common.GetProjectIDRolloutIDMaybeStageID(requestName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse rollout ID: %v", err))
	}
	rollout, err := s.store.GetRollout(ctx, rolloutID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get rollout: %v", err))
	}
	if rollout == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout %d not found", rolloutID))
	}

	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: &rollout.ID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get tasks: %v", err))
	}
	if len(tasks) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("rollout %d has no task", rollout.ID))
	}

	contents := []*exportData{}
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
		contents = append(contents, &exportData{
			Content:  exportArchive.Bytes,
			Database: task.GetDatabaseName(),
		})
	}

	encryptedBytes, err := doEncrypt(contents, &v1pb.ExportRequest{
		Password: tasks[0].Payload.GetPassword(),
		Format:   v1pb.ExportFormat(tasks[0].Payload.GetFormat()),
	})
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
	// only return the last result
	if len(results) > 1 {
		results = results[len(results)-1:]
	}
	if len(results) == 1 {
		if optionalAccessCheck != nil {
			if err := optionalAccessCheck(ctx, instance, database, user, spans, int(results[0].RowsCount), queryContext.Explain, true); err != nil {
				return nil, duration, err
			}
		}
	}

	if results[0].GetError() != "" {
		return nil, duration, errors.New(results[0].GetError())
	}

	if licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_DATA_MASKING, instance) == nil {
		masker := NewQueryResultMasker(stores)
		if err := masker.MaskResults(ctx, spans, results, instance, user, storepb.MaskingExceptionPolicy_MaskingException_EXPORT); err != nil {
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
		resourceList, err := getResources(ctx, stores, instance.Metadata.GetEngine(), database.DatabaseName, request.Statement, instance)
		if err != nil {
			return nil, 0, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to extract resource list: %v", err))
		}
		statementPrefix, err := getSQLStatementPrefix(instance.Metadata.GetEngine(), resourceList, result.ColumnNames)
		if err != nil {
			return nil, 0, err
		}
		content, err = exportSQL(instance.Metadata.GetEngine(), statementPrefix, result)
		if err != nil {
			return nil, duration, err
		}
	case v1pb.ExportFormat_XLSX:
		content, err = exportXLSX(result)
		if err != nil {
			return nil, duration, err
		}
	default:
		return nil, duration, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported export format: %s", request.Format.String()))
	}

	if request.Password == "" {
		return content, duration, nil
	}
	encryptedBytes, err := doEncrypt([]*exportData{
		{
			Database: database.DatabaseName,
			Content:  content,
		},
	}, request)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to encrypt data")
	}
	return encryptedBytes, duration, nil
}

type exportData struct {
	Database string
	Content  []byte
}

func doEncrypt(exports []*exportData, request *v1pb.ExportRequest) ([]byte, error) {
	var b bytes.Buffer
	fzip := io.Writer(&b)

	zipw := zip.NewWriter(fzip)
	defer zipw.Close()

	for i, export := range exports {
		fh := &zip.FileHeader{
			Name:   fmt.Sprintf("[%d] %s.%s", i, export.Database, strings.ToLower(request.Format.String())),
			Method: zip.Deflate,
		}
		fh.ModifiedDate, fh.ModifiedTime = timeToMsDosTime(time.Now())
		if request.Password != "" {
			fh.SetPassword(request.Password)
		}
		writer, err := zipw.CreateHeader(fh)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create encrypt export file")
		}
		if _, err := io.Copy(writer, bytes.NewReader(export.Content)); err != nil {
			return nil, errors.Wrapf(err, "failed to write export file")
		}
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
		return connect.NewError(connect.CodeInternal, errors.Errorf("Failed to create export history with error: %v", err))
	}
	return nil
}

func getListQueryHistoryFilter(filter string) (*store.ListResourceFilter, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
	}

	var getFilter func(expr celast.Expr) (string, error)
	var positionalArgs []any

	parseToSQL := func(variable, value any) (string, error) {
		switch variable {
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid project filter %q", value))
			}
			positionalArgs = append(positionalArgs, projectID)
			return fmt.Sprintf("query_history.project_id = $%d", len(positionalArgs)), nil
		case "database":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("query_history.database = $%d", len(positionalArgs)), nil
		case "instance":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("query_history.database LIKE $%d", len(positionalArgs)), nil
		case "type":
			historyType := store.QueryHistoryType(value.(string))
			positionalArgs = append(positionalArgs, historyType)
			return fmt.Sprintf("query_history.type = $%d", len(positionalArgs)), nil
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q", variable))
		}
	}

	getFilter = func(expr celast.Expr) (string, error) {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				return getSubConditionFromExpr(expr, getFilter, "OR")
			case celoperators.LogicalAnd:
				return getSubConditionFromExpr(expr, getFilter, "AND")
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				return parseToSQL(variable, value)
			case celoverloads.Matches:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`invalid args for %q`, variable))
				}
				value := args[0].AsLiteral().Value()
				if variable != "statement" {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`only "statement" support %q operator, but found %q`, celoverloads.Matches, variable))
				}
				strValue, ok := value.(string)
				if !ok {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("expect string, got %T, hint: filter literals should be string", value))
				}
				return "query_history.statement LIKE '%" + strValue + "%'", nil
			default:
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected function %v", functionName))
			}
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected expr kind %v", expr.Kind()))
		}
	}

	where, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, err
	}

	return &store.ListResourceFilter{
		Args:  positionalArgs,
		Where: "(" + where + ")",
	}, nil
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

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("principal ID not found"))
	}

	find := &store.FindQueryHistoryMessage{
		CreatorUID: &principalID,
		Limit:      &limitPlusOne,
		Offset:     &offset.offset,
	}
	filter, err := getListQueryHistoryFilter(request.Filter)
	if err != nil {
		return nil, err
	}
	find.Filter = filter

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
	default:
	}

	return &v1pb.QueryHistory{
		Name:       fmt.Sprintf("queryHistories/%d", history.UID),
		Statement:  history.Statement,
		Error:      history.Payload.Error,
		Database:   history.Database,
		Creator:    common.FormatUserEmail(user.Email),
		CreateTime: timestamppb.New(history.CreatedAt),
		Duration:   history.Payload.Duration,
		Type:       historyType,
	}, nil
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
			meta, err := storeInstance.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
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
		linkedDatabaseMetadata, err := storeInstance.GetDBSchema(ctx, linkedDatabase.InstanceID, linkedDatabase.DatabaseName)
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
		databaseMetadata, err := storeInstance.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
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
	limit int,
	isExplain bool,
	isExport bool) error {
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
			for column := range span.SourceColumns {
				attributes := map[string]any{
					"request.time":      time.Now(),
					"request.row_limit": limit,
					"resource.database": common.FormatDatabase(instance.ResourceID, column.Database),
					"resource.schema":   column.Schema,
					"resource.table":    column.Table,
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

				ok, err := s.hasDatabaseAccessRights(ctx, user, []*storepb.IamPolicy{workspacePolicy.Policy, projectPolicy.Policy}, attributes, isExport)
				if err != nil {
					return connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access control for database: %q, error %v", column.Database, err))
				}
				if !ok {
					resource := attributes["resource.database"]
					if schema, ok := attributes["resource.schema"]; ok && schema != "" {
						resource = fmt.Sprintf("%s/schemas/%s", resource, schema)
					}
					if table, ok := attributes["resource.table"]; ok && table != "" {
						resource = fmt.Sprintf("%s/tables/%s", resource, table)
					}
					return connect.NewError(
						connect.CodePermissionDenied,
						errors.Errorf("permission denied to access resource: %s", resource),
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
				Code:    int32(advisor.StatementSyntaxError),
				Content: syntaxErr.Message,
				Title:   "Syntax error",
				Status:  v1pb.PlanCheckRun_Result_ERROR,
				Report: &v1pb.PlanCheckRun_Result_SqlReviewReport_{
					SqlReviewReport: &v1pb.PlanCheckRun_Result_SqlReviewReport{
						Line:   int32(syntaxErr.Position.GetLine()),
						Column: int32(syntaxErr.Position.GetColumn()),
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
		switch instance.Metadata.GetEngine() {
		case storepb.Engine_REDIS, storepb.Engine_MONGODB:
			return nonReadOnlyCommandError
		default:
			return nonSelectSQLError
		}
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
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("the user has been deactivated"))
	}
	return user, nil
}

func (s *SQLService) Check(ctx context.Context, req *connect.Request[v1pb.CheckRequest]) (*connect.Response[v1pb.CheckResponse], error) {
	request := req.Msg
	if len(request.Statement) > common.MaxSheetCheckSize {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("statement size exceeds maximum allowed size %dKB", common.MaxSheetCheckSize/1024))
	}

	database, err := getDatabaseMessage(ctx, s.store, request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New(err.Error()))
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &database.InstanceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get instance, error: %v", err))
	}
	if instance == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", database.InstanceID))
	}

	checkResponse := &v1pb.CheckResponse{}
	changeType := convertChangeType(request.ChangeType)
	// Get SQL summary report for the statement and target database.
	// Including affected rows.
	summaryReport, err := plancheck.GetSQLSummaryReport(ctx, s.store, s.sheetManager, s.dbFactory, database, request.Statement)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get SQL summary report, error: %v", err))
	}
	if summaryReport != nil {
		checkResponse.AffectedRows = summaryReport.AffectedRows
	}

	_, adviceList, err := s.SQLReviewCheck(ctx, request.Statement, changeType, instance, database)
	if err != nil {
		return nil, err
	}
	checkResponse.Advices = adviceList
	return connect.NewResponse(checkResponse), nil
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

// SQLReviewCheck checks the SQL statement against the SQL review policy bind to given environment.
func (s *SQLService) SQLReviewCheck(
	ctx context.Context,
	statement string,
	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
) (storepb.Advice_Status, []*v1pb.Advice, error) {
	if !common.EngineSupportSQLReview(instance.Metadata.GetEngine()) || database == nil {
		return storepb.Advice_SUCCESS, nil, nil
	}

	dbSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
	if err != nil {
		return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to fetch database schema for database %s", database.String())
	}
	if dbSchema == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to sync database schema for database %s", database.String())
		}
		dbSchema, err = s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
		if err != nil {
			return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to fetch database schema for database %s", database.String())
		}
		if dbSchema == nil {
			return storepb.Advice_ERROR, nil, errors.Wrapf(err, "cannot found schema for database %s", database.String())
		}
	}
	dbMetadata := dbSchema.GetMetadata()

	catalog, err := catalog.NewCatalog(ctx, s.store, database.InstanceID, database.DatabaseName, instance.Metadata.GetEngine(), store.IsObjectCaseSensitive(instance), dbMetadata)
	if err != nil {
		return storepb.Advice_ERROR, nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create a catalog: %v", err))
	}

	useDatabaseOwner, err := getUseDatabaseOwner(ctx, s.store, instance, database, changeType)
	if err != nil {
		return storepb.Advice_ERROR, nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get use database owner: %v", err))
	}
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{UseDatabaseOwner: useDatabaseOwner})
	if err != nil {
		return storepb.Advice_ERROR, nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get database driver: %v", err))
	}
	defer driver.Close(ctx)
	connection := driver.GetDB()

	classificationConfig := getClassificationByProject(ctx, s.store, database.ProjectID)
	context := advisor.SQLReviewCheckContext{
		Charset:                  dbMetadata.CharacterSet,
		Collation:                dbMetadata.Collation,
		ChangeType:               changeType,
		DBSchema:                 dbMetadata,
		DBType:                   instance.Metadata.GetEngine(),
		Catalog:                  catalog,
		Driver:                   connection,
		CurrentDatabase:          database.DatabaseName,
		ClassificationConfig:     classificationConfig,
		UsePostgresDatabaseOwner: useDatabaseOwner,
		ListDatabaseNamesFunc:    BuildListDatabaseNamesFunc(s.store),
		InstanceID:               instance.ResourceID,
		IsObjectCaseSensitive:    store.IsObjectCaseSensitive(instance),
	}

	reviewConfig, err := s.store.GetReviewConfigForDatabase(ctx, database)
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			// Continue to check the builtin rules.
			reviewConfig = &storepb.ReviewConfigPayload{}
		} else {
			return storepb.Advice_ERROR, nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get SQL review policy with error: %v", err))
		}
	}

	res, err := advisor.SQLReviewCheck(ctx, s.sheetManager, statement, reviewConfig.SqlReviewRules, context)
	if err != nil {
		return storepb.Advice_ERROR, nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to exec SQL review with error: %v", err))
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
		default:
		}

		advices = append(advices, convertToV1Advice(advice))
	}

	return adviceLevel, advices, nil
}

func getUseDatabaseOwner(ctx context.Context, stores *store.Store, instance *store.InstanceMessage, database *store.DatabaseMessage, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) (bool, error) {
	if instance.Metadata.GetEngine() != storepb.Engine_POSTGRES || changeType == storepb.PlanCheckRunConfig_SQL_EDITOR {
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

// Pretty returns pretty format SDL.
func (*SQLService) Pretty(_ context.Context, req *connect.Request[v1pb.PrettyRequest]) (*connect.Response[v1pb.PrettyResponse], error) {
	request := req.Msg
	engine := convertEngine(request.Engine)
	if _, err := transform.CheckFormat(engine, request.ExpectedSchema); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("User SDL is not SDL format: %s", err.Error()))
	}
	if _, err := transform.CheckFormat(engine, request.CurrentSchema); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("Dumped SDL is not SDL format: %s", err.Error()))
	}

	prettyExpectedSchema, err := transform.SchemaTransform(engine, request.ExpectedSchema)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to transform user SDL: %s", err.Error()))
	}
	prettyCurrentSchema, err := transform.Normalize(engine, request.CurrentSchema, prettyExpectedSchema)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to normalize dumped SDL: %s", err.Error()))
	}

	return connect.NewResponse(&v1pb.PrettyResponse{
		CurrentSchema:  prettyCurrentSchema,
		ExpectedSchema: prettyExpectedSchema,
	}), nil
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
				return connect.NewError(connect.CodePermissionDenied, errors.Errorf("disallow execute DDL statement in environment %q", environment.Title))
			}
		case parserbase.DML:
			if policy.DisallowDml {
				return connect.NewError(connect.CodePermissionDenied, errors.Errorf("disallow execute DML statement in environment %q", environment.Title))
			}
		default:
		}
	}
	return nil
}
