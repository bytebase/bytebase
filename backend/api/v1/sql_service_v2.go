package v1

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/masker"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (s *SQLService) ExportV2(ctx context.Context, request *v1pb.ExportRequest) (*v1pb.ExportResponse, error) {
	// Prepare related message.
	user, environment, instance, maybeDatabase, err := s.prepareRelatedMessage(ctx, request.Name, request.ConnectionDatabase)
	if err != nil {
		return nil, err
	}

	statement := request.Statement
	// In Redshift datashare, Rewrite query used for parser.
	if maybeDatabase != nil && maybeDatabase.DataShare {
		statement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", maybeDatabase.DatabaseName), "")
	}

	// Validate the request.
	if err := validateQueryRequest(instance, request.ConnectionDatabase, statement); err != nil {
		return nil, err
	}

	schemaName := ""
	if instance.Engine == storepb.Engine_ORACLE {
		// For Oracle, there are two modes, schema-based and database-based management.
		// For schema-based management, also say tenant mode, we need to use the schemaName as the databaseName.
		// So the default schemaName is the connectionDatabase.
		// For database-based management, we need to use the dataSource.Username as the schemaName.
		// So the default schemaName is the dataSource.Username.
		isSchemaTenantMode := (instance.Options != nil && instance.Options.GetSchemaTenantMode())
		if isSchemaTenantMode {
			schemaName = request.ConnectionDatabase
		} else {
			dataSource, _, err := s.dbFactory.GetReadOnlyDatabaseSource(instance, maybeDatabase, "" /* dataSourceID */)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get read only database source")
			}
			schemaName = dataSource.Username
		}
	}

	spans, err := base.GetQuerySpan(
		ctx,
		instance.Engine,
		statement,
		request.ConnectionDatabase,
		schemaName,
		s.buildGetDatabaseMetadataFunc(instance, request.ConnectionDatabase),
		s.buildListDatabaseNamesFunc(instance),
		store.IgnoreDatabaseAndTableCaseSensitive(instance),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span")
	}

	if s.licenseService.IsFeatureEnabled(api.FeatureAccessControl) == nil {
		if err := s.accessCheck(ctx, instance, environment, user, request.Statement, spans, request.Limit, false /* isAdmin */, true /* isExport */); err != nil {
			return nil, err
		}
	}

	// Run SQL review.
	if _, _, err = s.sqlReviewCheck(ctx, statement, v1pb.CheckRequest_CHANGE_TYPE_UNSPECIFIED, environment, instance, maybeDatabase, nil /* Override Metadata */); err != nil {
		return nil, err
	}

	databaseID := 0
	if maybeDatabase != nil {
		databaseID = maybeDatabase.UID
	}
	// Create export activity.
	activity, err := s.createExportActivity(ctx, user, api.ActivityInfo, instance.UID, api.ActivitySQLExportPayload{
		Statement:    request.Statement,
		InstanceID:   instance.UID,
		DatabaseID:   databaseID,
		DatabaseName: request.ConnectionDatabase,
	})
	if err != nil {
		return nil, err
	}

	bytes, durationNs, exportErr := s.doExportV2(ctx, request, instance, maybeDatabase, spans)

	if err := s.postExport(ctx, activity, durationNs, exportErr); err != nil {
		return nil, err
	}

	if exportErr != nil {
		return nil, exportErr
	}

	content, err := doEncrypt(bytes, request)
	if err != nil {
		return nil, err
	}

	return &v1pb.ExportResponse{
		Content: content,
	}, nil
}

func (s *SQLService) QueryV2(ctx context.Context, request *v1pb.QueryRequest) (*v1pb.QueryResponse, error) {
	// Prepare related message.
	user, environment, instance, maybeDatabase, err := s.prepareRelatedMessage(ctx, request.Name, request.ConnectionDatabase)
	if err != nil {
		return nil, err
	}

	statement := request.Statement
	// In Redshift datashare, Rewrite query used for parser.
	if maybeDatabase != nil && maybeDatabase.DataShare {
		statement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", maybeDatabase.DatabaseName), "")
	}

	// Validate the request.
	if err := validateQueryRequest(instance, request.ConnectionDatabase, statement); err != nil {
		return nil, err
	}

	schemaName := ""
	if instance.Engine == storepb.Engine_ORACLE {
		// For Oracle, there are two modes, schema-based and database-based management.
		// For schema-based management, also say tenant mode, we need to use the schemaName as the databaseName.
		// So the default schemaName is the connectionDatabase.
		// For database-based management, we need to use the dataSource.Username as the schemaName.
		// So the default schemaName is the dataSource.Username.
		isSchemaTenantMode := (instance.Options != nil && instance.Options.GetSchemaTenantMode())
		if isSchemaTenantMode {
			schemaName = request.ConnectionDatabase
		} else {
			dataSource, _, err := s.dbFactory.GetReadOnlyDatabaseSource(instance, maybeDatabase, "" /* dataSourceID */)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get read only database source")
			}
			schemaName = dataSource.Username
		}
	}

	// Get query span.
	spans, err := base.GetQuerySpan(
		ctx,
		instance.Engine,
		statement,
		request.ConnectionDatabase,
		schemaName,
		s.buildGetDatabaseMetadataFunc(instance, request.ConnectionDatabase),
		s.buildListDatabaseNamesFunc(instance),
		store.IgnoreDatabaseAndTableCaseSensitive(instance),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span")
	}

	if s.licenseService.IsFeatureEnabled(api.FeatureAccessControl) == nil {
		if err := s.accessCheck(ctx, instance, environment, user, request.Statement, spans, request.Limit, false /* isAdmin */, false /* isExport */); err != nil {
			return nil, err
		}
	}

	// Run SQL review.
	adviceStatus, advices, err := s.sqlReviewCheck(ctx, statement, v1pb.CheckRequest_CHANGE_TYPE_UNSPECIFIED, environment, instance, maybeDatabase, nil /* Override Metadata */)
	if err != nil {
		return nil, err
	}
	// Create query activity.
	level := api.ActivityInfo
	switch adviceStatus {
	case advisor.Error:
		level = api.ActivityError
	case advisor.Warn:
		level = api.ActivityWarn
	}

	databaseID := 0
	if maybeDatabase != nil {
		databaseID = maybeDatabase.UID
	}
	activity, err := s.createQueryActivity(ctx, user, level, instance.UID, api.ActivitySQLEditorQueryPayload{
		Statement:              request.Statement,
		InstanceID:             instance.UID,
		DeprecatedInstanceName: instance.Title,
		DatabaseID:             databaseID,
		DatabaseName:           request.ConnectionDatabase,
	})
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	var queryErr error
	var durationNs int64
	if adviceStatus != advisor.Error {
		results, durationNs, queryErr = s.doQueryV2(ctx, request, instance, maybeDatabase)
		if queryErr == nil && s.licenseService.IsFeatureEnabledForInstance(api.FeatureSensitiveData, instance) == nil {
			if err := s.maskResults(ctx, spans, results, instance, storepb.MaskingExceptionPolicy_MaskingException_QUERY); err != nil {
				return nil, err
			}
		}
	}

	// Update activity.
	err = s.postQuery(ctx, activity, durationNs, queryErr)
	if err != nil {
		return nil, err
	}
	if queryErr != nil {
		return nil, queryErr
	}

	allowExport := true
	// AllowExport is a validate only check.
	if s.licenseService.IsFeatureEnabled(api.FeatureAccessControl) == nil {
		err := s.accessCheck(ctx, instance, environment, user, request.Statement, spans, request.Limit, false /* isAdmin */, true /* isExport */)
		allowExport = (err == nil)
	}

	response := &v1pb.QueryResponse{
		Results:     results,
		Advices:     advices,
		AllowExport: allowExport,
	}

	if proto.Size(response) > maximumSQLResultSize {
		response.Results = []*v1pb.QueryResult{
			{
				Error: fmt.Sprintf("Output of query exceeds max allowed output size of %dMB", maximumSQLResultSize/1024/1024),
			},
		}
	}

	return response, nil
}

// doExportV2 is the copy of doExport, which use query span to improve performance.
func (s *SQLService) doExportV2(ctx context.Context, request *v1pb.ExportRequest, instance *store.InstanceMessage, database *store.DatabaseMessage, spans []*base.QuerySpan) ([]byte, int64, error) {
	driver, err := s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, database, "" /* dataSourceID */)
	if err != nil {
		return nil, 0, err
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

	start := time.Now().UnixNano()
	result, err := driver.QueryConn(ctx, conn, request.Statement, &db.QueryContext{
		Limit:               int(request.Limit),
		ReadOnly:            true,
		CurrentDatabase:     request.ConnectionDatabase,
		SensitiveSchemaInfo: nil,
		EnableSensitive:     s.licenseService.IsFeatureEnabledForInstance(api.FeatureSensitiveData, instance) == nil,
	})
	durationNs := time.Now().UnixNano() - start
	if err != nil {
		return nil, durationNs, err
	}
	// only return the last result
	if len(result) > 1 {
		result = result[len(result)-1:]
	}

	if s.licenseService.IsFeatureEnabledForInstance(api.FeatureSensitiveData, instance) == nil {
		if err := s.maskResults(ctx, spans, result, instance, storepb.MaskingExceptionPolicy_MaskingException_EXPORT); err != nil {
			return nil, durationNs, err
		}
	}

	var content []byte
	switch request.Format {
	case v1pb.ExportFormat_CSV:
		if content, err = exportCSV(result[0]); err != nil {
			return nil, durationNs, err
		}
	case v1pb.ExportFormat_JSON:
		if content, err = exportJSON(result[0]); err != nil {
			return nil, durationNs, err
		}
	case v1pb.ExportFormat_SQL:
		resourceList, err := s.extractResourceList(ctx, instance.Engine, request.ConnectionDatabase, request.Statement, instance)
		if err != nil {
			return nil, 0, status.Errorf(codes.InvalidArgument, "failed to extract resource list: %v", err)
		}
		statementPrefix, err := getSQLStatementPrefix(instance.Engine, resourceList, result[0].ColumnNames)
		if err != nil {
			return nil, 0, err
		}
		if content, err = exportSQL(instance.Engine, statementPrefix, result[0]); err != nil {
			return nil, durationNs, err
		}
	case v1pb.ExportFormat_XLSX:
		if content, err = exportXLSX(result[0]); err != nil {
			return nil, durationNs, err
		}
	default:
		return nil, durationNs, status.Errorf(codes.InvalidArgument, "unsupported export format: %s", request.Format.String())
	}
	return content, durationNs, nil
}

// doQueryV2 is the copy of doQuery, which use query span to improve performance.
func (s *SQLService) doQueryV2(ctx context.Context, request *v1pb.QueryRequest, instance *store.InstanceMessage, database *store.DatabaseMessage) ([]*v1pb.QueryResult, int64, error) {
	driver, err := s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, database, request.DataSourceId)
	if err != nil {
		return nil, 0, err
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

	timeout := defaultTimeout
	if request.Timeout != nil {
		timeout = request.Timeout.AsDuration()
	}
	ctx, cancelCtx := context.WithTimeout(ctx, timeout)
	defer cancelCtx()

	start := time.Now().UnixNano()
	results, err := driver.QueryConn(ctx, conn, request.Statement, &db.QueryContext{
		Limit:               int(request.Limit),
		ReadOnly:            true,
		CurrentDatabase:     request.ConnectionDatabase,
		SensitiveSchemaInfo: nil,
		EnableSensitive:     s.licenseService.IsFeatureEnabledForInstance(api.FeatureSensitiveData, instance) == nil,
	})
	select {
	case <-ctx.Done():
		// canceled or timed out
		return nil, time.Now().UnixNano() - start, errors.Errorf("timeout reached: %v", timeout)
	default:
		// So the select will not block
	}

	sanitizeResults(results)

	return results, time.Now().UnixNano() - start, err
}

func (s *SQLService) getMasterForColumnResource(
	ctx context.Context,
	m *maskingLevelEvaluator,
	instance *store.InstanceMessage,
	sourceColumn base.ColumnResource,
	maskingExceptionPolicyMap map[string]*storepb.MaskingExceptionPolicy,
	action storepb.MaskingExceptionPolicy_MaskingException_Action,
	currentPrincipal *store.UserMessage,
) (masker.Masker, error) {
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instance.ResourceID,
		DatabaseName: &sourceColumn.Database,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find database: %q", sourceColumn.Database)
	}
	if database == nil {
		return masker.NewNoneMasker(), nil
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &database.ProjectID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find project: %q", database.ProjectID)
	}
	if project == nil {
		return masker.NewNoneMasker(), nil
	}

	meta, config, err := s.getColumnForColumnResource(ctx, instance.ResourceID, &sourceColumn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for column resource: %q", sourceColumn.String())
	}
	// Span and metadata are not the same in real time, so we fall back to none masker.
	if meta == nil {
		return masker.NewNoneMasker(), nil
	}

	semanticTypeID := ""
	if config != nil {
		semanticTypeID = config.SemanticTypeId
	}

	maskingPolicy, err := s.store.GetMaskingPolicyByDatabaseUID(ctx, database.UID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get masking policy for database: %q", database.DatabaseName)
	}
	maskingPolicyMap := make(map[maskingPolicyKey]*storepb.MaskData)
	if maskingPolicy != nil {
		for _, maskData := range maskingPolicy.MaskData {
			maskingPolicyMap[maskingPolicyKey{
				schema: maskData.Schema,
				table:  maskData.Table,
				column: maskData.Column,
			}] = maskData
		}
	}

	var maskingExceptionPolicy *storepb.MaskingExceptionPolicy
	// If we cannot find the maskingExceptionPolicy before, we need to find it from the database and record it in cache.

	if _, ok := maskingExceptionPolicyMap[database.ProjectID]; !ok {
		policy, err := s.store.GetMaskingExceptionPolicyByProjectUID(ctx, project.UID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find masking exception policy for project %q", project.ResourceID)
		}
		// It is safe if policy is nil.
		maskingExceptionPolicyMap[database.ProjectID] = policy
	}
	maskingExceptionPolicy = maskingExceptionPolicyMap[database.ProjectID]

	// Build the filtered maskingExceptionPolicy for current principal.
	var maskingExceptionContainsCurrentPrincipal []*storepb.MaskingExceptionPolicy_MaskingException
	if maskingExceptionPolicy != nil {
		for _, maskingException := range maskingExceptionPolicy.MaskingExceptions {
			if maskingException.Action != action {
				continue
			}
			if maskingException.Member == currentPrincipal.Email {
				maskingExceptionContainsCurrentPrincipal = append(maskingExceptionContainsCurrentPrincipal, maskingException)
			}
		}
	}

	maskingAlgorithm, maskingLevel, err := m.evaluateMaskingAlgorithmOfColumn(database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column, semanticTypeID, meta.Classification, project.DataClassificationConfigID, maskingPolicyMap, maskingExceptionContainsCurrentPrincipal)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to evaluate masking level of database %q, schema %q, table %q, column %q", sourceColumn.Database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column)
	}
	return getMaskerByMaskingAlgorithmAndLevel(maskingAlgorithm, maskingLevel), nil
}

// getMaskersForQuerySpan returns the maskers for the query span.
func (s *SQLService) getMaskersForQuerySpan(ctx context.Context, m *maskingLevelEvaluator, instance *store.InstanceMessage, span *base.QuerySpan, action storepb.MaskingExceptionPolicy_MaskingException_Action) ([]masker.Masker, error) {
	if span == nil {
		return nil, nil
	}
	maskers := make([]masker.Masker, 0, len(span.Results))

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	currentPrincipal, err := s.store.GetUser(ctx, &store.FindUserMessage{
		ID: &principalID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find current principal")
	}
	if currentPrincipal == nil {
		return nil, status.Errorf(codes.Internal, "current principal not found")
	}

	// Multiple databases may belong to the same project, to reduce the protojson unmarshal cost,
	// we store the projectResourceID - maskingExceptionPolicy in a map.
	maskingExceptionPolicyMap := make(map[string]*storepb.MaskingExceptionPolicy)

	for _, spanResult := range span.Results {
		// Likes constant expression, we use the none masker.
		if len(spanResult.SourceColumns) == 0 {
			maskers = append(maskers, masker.NewNoneMasker())
			continue
		}

		var effectiveMaskers []masker.Masker
		for column := range spanResult.SourceColumns {
			newMasker, err := s.getMasterForColumnResource(ctx, m, instance, column, maskingExceptionPolicyMap, action, currentPrincipal)
			if err != nil {
				return nil, err
			}
			if newMasker == nil {
				continue
			}
			if _, ok := newMasker.(*masker.NoneMasker); ok {
				continue
			}
			effectiveMaskers = append(effectiveMaskers, newMasker)
		}

		switch len(effectiveMaskers) {
		case 0:
			maskers = append(maskers, masker.NewNoneMasker())
		case 1:
			maskers = append(maskers, effectiveMaskers[0])
		default:
			// If there are more than one source columns, we fall back to the default full masker,
			// because we don't know how the data be made up.
			maskers = append(maskers, masker.NewDefaultFullMasker())
		}
	}
	return maskers, nil
}

func (s *SQLService) getColumnForColumnResource(ctx context.Context, instanceID string, sourceColumn *base.ColumnResource) (*storepb.ColumnMetadata, *storepb.ColumnConfig, error) {
	if sourceColumn == nil {
		return nil, nil, nil
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &sourceColumn.Database,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find database: %q", sourceColumn.Database)
	}
	if database == nil {
		return nil, nil, nil
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find database schema: %q", sourceColumn.Database)
	}
	if dbSchema == nil {
		return nil, nil, nil
	}

	var columnMetadata *storepb.ColumnMetadata
	metadata := dbSchema.GetDatabaseMetadata()
	if metadata == nil {
		return nil, nil, nil
	}
	schema := metadata.GetSchema(sourceColumn.Schema)
	if schema == nil {
		return nil, nil, nil
	}
	table := schema.GetTable(sourceColumn.Table)
	if table == nil {
		return nil, nil, nil
	}
	column := table.GetColumn(sourceColumn.Column)
	if column == nil {
		return nil, nil, nil
	}
	columnMetadata = column

	var columnConfig *storepb.ColumnConfig
	config := dbSchema.GetDatabaseConfig()
	if config == nil {
		return columnMetadata, nil, nil
	}
	schemaConfig := config.GetSchemaConfig(sourceColumn.Schema)
	if schemaConfig == nil {
		return columnMetadata, nil, nil
	}
	tableConfig := schemaConfig.GetTableConfig(sourceColumn.Table)
	if tableConfig == nil {
		return columnMetadata, nil, nil
	}

	columnConfig = tableConfig.GetColumnConfig(sourceColumn.Column)
	return columnMetadata, columnConfig, nil
}

func (s *SQLService) buildGetDatabaseMetadataFunc(instance *store.InstanceMessage, connectionDatabase string) base.GetDatabaseMetadataFunc {
	if instance.Engine == storepb.Engine_ORACLE {
		return func(ctx context.Context, schemaName string) (string, *model.DatabaseMetadata, error) {
			// There are two modes for Oracle, schema-based and database-based management.
			// For schema-based management, also say tenant mode, we need to use the schemaName as the databaseName.
			// For database-based management, we need to use the connectionDatabase as the databaseName.
			databaseName := connectionDatabase
			isSchemaTenantMode := (instance.Options != nil && instance.Options.GetSchemaTenantMode())
			if isSchemaTenantMode {
				databaseName = schemaName
			}

			database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				InstanceID:   &instance.ResourceID,
				DatabaseName: &databaseName,
			})
			if err != nil {
				return "", nil, err
			}
			if database == nil {
				return "", nil, nil
			}
			databaseMetadata, err := s.store.GetDBSchema(ctx, database.UID)
			if err != nil {
				return "", nil, err
			}
			if databaseMetadata == nil {
				return "", nil, nil
			}
			return databaseName, databaseMetadata.GetDatabaseMetadata(), nil
		}
	}
	return func(ctx context.Context, databaseName string) (string, *model.DatabaseMetadata, error) {
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instance.ResourceID,
			DatabaseName: &databaseName,
		})
		if err != nil {
			return "", nil, err
		}
		if database == nil {
			return "", nil, nil
		}
		databaseMetadata, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return "", nil, err
		}
		if databaseMetadata == nil {
			return "", nil, nil
		}
		return databaseName, databaseMetadata.GetDatabaseMetadata(), nil
	}
}

func (s *SQLService) buildListDatabaseNamesFunc(instance *store.InstanceMessage) base.ListDatabaseNamesFunc {
	return func(ctx context.Context) ([]string, error) {
		databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{
			InstanceID: &instance.ResourceID,
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
	environment *store.EnvironmentMessage,
	user *store.UserMessage,
	statement string,
	spans []*base.QuerySpan,
	limit int32,
	isAdmin,
	isExport bool) error {
	// Check if the caller is admin for exporting with admin mode.
	if isAdmin && isExport && (user.Role != api.WorkspaceAdmin && user.Role != api.WorkspaceDBA) {
		return status.Errorf(codes.PermissionDenied, "only workspace owner and DBA can export data using admin mode")
	}

	// Check if the environment is open for query privileges.
	ok, err := s.checkWorkspaceIAMPolicy(ctx, environment, isExport)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	for _, span := range spans {
		for column := range span.SourceColumns {
			databaseResourceURL := common.FormatDatabase(instance.ResourceID, column.Database)
			attributes := map[string]any{
				"request.time":      time.Now(),
				"resource.database": databaseResourceURL,
				"resource.schema":   column.Schema,
				"resource.table":    column.Table,
				"request.statement": encodeToBase64String(statement),
				"request.row_limit": limit,
			}

			project, database, err := s.getProjectAndDatabaseMessage(ctx, instance, column.Database)
			if err != nil {
				return err
			}
			if project == nil && database == nil {
				// If database not found, skip.
				// TODO(d): re-evaluate this case.
				continue
			}
			if project == nil {
				// Never happen
				return status.Errorf(codes.Internal, "project not found for database: %s", column.Database)
			}
			// Allow query databases across different projects.
			projectPolicy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
			if err != nil {
				return err
			}

			ok, err := s.hasDatabaseAccessRights(ctx, user, projectPolicy, attributes, isExport)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to check access control for database: %q, error %v", column.Database, err)
			}
			if !ok {
				return status.Errorf(codes.PermissionDenied, "permission denied to access resource: %q", column.String())
			}
		}
	}

	return nil
}

// maskResult masks the result in-place based on the dynamic masking policy, query-span, instance and action.
func (s *SQLService) maskResults(ctx context.Context, spans []*base.QuerySpan, results []*v1pb.QueryResult, instance *store.InstanceMessage, action storepb.MaskingExceptionPolicy_MaskingException_Action) error {
	classificationSetting, err := s.store.GetDataClassificationSetting(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find classification setting")
	}

	maskingRulePolicy, err := s.store.GetMaskingRulePolicy(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find masking rule policy")
	}

	algorithmSetting, err := s.store.GetMaskingAlgorithmSetting(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find masking algorithm setting")
	}

	semanticTypesSetting, err := s.store.GetSemanticTypesSetting(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find semantic types setting")
	}

	m := newEmptyMaskingLevelEvaluator().
		withMaskingRulePolicy(maskingRulePolicy).
		withDataClassificationSetting(classificationSetting).
		withMaskingAlgorithmSetting(algorithmSetting).
		withSemanticTypeSetting(semanticTypesSetting)

	// We expect the len(spans) == len(results), but to avoid NPE, we use the min(len(spans), len(results)) here.
	loopBoundary := min(len(spans), len(results))
	for i := 0; i < loopBoundary; i++ {
		maskers, err := s.getMaskersForQuerySpan(ctx, m, instance, spans[i], action)
		if err != nil {
			return errors.Wrapf(err, "failed to get maskers for query span")
		}
		mask(maskers, results[i])
	}

	return nil
}

func mask(maskers []masker.Masker, result *v1pb.QueryResult) {
	sensitive := make([]bool, len(result.ColumnNames))
	for i := range result.ColumnNames {
		if i < len(maskers) {
			switch maskers[i].(type) {
			case *masker.NoneMasker:
				sensitive[i] = false
			default:
				sensitive[i] = true
			}
		}
	}

	for i, row := range result.Rows {
		for j, value := range row.Values {
			if value == nil {
				continue
			}
			maskedValue := row.Values[j]
			if j < len(maskers) && maskers[j] != nil {
				maskedValue = maskers[j].Mask(&masker.MaskData{
					DataV2: row.Values[j],
				})
			}
			result.Rows[i].Values[j] = maskedValue
		}
	}

	result.Sensitive = sensitive
	result.Masked = sensitive
}
