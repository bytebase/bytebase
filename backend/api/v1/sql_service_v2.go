package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

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

	// Get query span.
	span, err := base.GetQuerySpan(ctx, instance.Engine, statement, s.buildGetDatabaseMetadataFunc(instance))
	if err != nil {
		return nil, err
	}

	if s.licenseService.IsFeatureEnabled(api.FeatureAccessControl) == nil {
		if err := s.accessCheck(ctx, instance, environment, user, request.Statement, span, request.Limit, false /* isAdmin */, false /* isExport */); err != nil {
			return nil, err
		}
	}

	// Run SQL review.
	adviceStatus, advices, err := s.sqlReviewCheck(ctx, statement, environment, instance, maybeDatabase)
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
		results, durationNs, queryErr = s.doQuery(ctx, request, instance, maybeDatabase, nil)
	}

	// Update activity.
	err = s.postQuery(ctx, activity, durationNs, queryErr)
	if err != nil {
		return nil, err
	}
	if queryErr != nil {
		return nil, queryErr
	}

	// Return response.
	response := &v1pb.QueryResponse{
		Results: results,
		Advices: advices,
		// AllowExport: allowExport,
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

func (s *SQLService) buildGetDatabaseMetadataFunc(instance *store.InstanceMessage) base.GetDatabaseMetadataFunc {
	return func(ctx context.Context, databaseName string) (*model.DatabaseMetadata, error) {
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instance.ResourceID,
			DatabaseName: &databaseName,
		})
		if err != nil {
			return nil, err
		}
		databaseMetadata, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, err
		}
		return databaseMetadata.GetDatabaseMetadata(), nil
	}
}

func (s *SQLService) accessCheck(
	ctx context.Context,
	instance *store.InstanceMessage,
	environment *store.EnvironmentMessage,
	user *store.UserMessage,
	statement string,
	span *base.QuerySpan,
	limit int32,
	isAdmin,
	isExport bool) error {
	// Check if the caller is admin for exporting with admin mode.
	if isAdmin && (user.Role != api.Owner && user.Role != api.DBA) {
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

	for column, _ := range span.SourceColumns {
		databaseResourceURL := fmt.Sprintf("instances/%s/databases/%s", instance.ResourceID, column.Database)
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

		ok, err := hasDatabaseAccessRights(user.ID, projectPolicy, attributes, isExport)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check access control for database: %q", column.Database)
		}
		if !ok {
			return status.Errorf(codes.PermissionDenied, "permission denied to access resource: %q", column.String())
		}
	}

	return nil
}
