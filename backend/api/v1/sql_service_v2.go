package v1

import (
	"context"
	"fmt"

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

	// Validate the request.
	if err := validateQueryRequest(instance, request.ConnectionDatabase, request.Statement); err != nil {
		return nil, err
	}

	// Get query span.
	_, err = base.GetQuerySpan(ctx, instance.Engine, request.Statement, s.buildGetDatabaseMetadataFunc(instance))
	if err != nil {
		return nil, err
	}

	// Run SQL review.
	adviceStatus, advices, err := s.sqlReviewCheck(ctx, request.Statement, environment, instance, maybeDatabase)
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
