package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (in *ACLInterceptor) checkIAMPermission(ctx context.Context, fullMethod string, req any, user *store.UserMessage) error {
	p, ok := methodPermissionMap[fullMethod]
	if !ok {
		return nil
	}

	switch fullMethod {
	// handled in the method because checking is complex.
	case
		v1pb.DatabaseService_ListDatabases_FullMethodName:

	// below are "workspace-level" permissions.
	// we don't have to go down to the project level.
	case
		v1pb.InstanceService_ListInstances_FullMethodName,
		v1pb.InstanceService_GetInstance_FullMethodName,
		v1pb.InstanceService_CreateInstance_FullMethodName,
		v1pb.InstanceService_UpdateInstance_FullMethodName,
		v1pb.InstanceService_DeleteInstance_FullMethodName,
		v1pb.InstanceService_UndeleteInstance_FullMethodName,
		v1pb.InstanceService_SyncInstance_FullMethodName,
		v1pb.InstanceService_BatchSyncInstance_FullMethodName,
		v1pb.InstanceService_AddDataSource_FullMethodName,
		v1pb.InstanceService_RemoveDataSource_FullMethodName,
		v1pb.InstanceService_UpdateDataSource_FullMethodName,
		v1pb.InstanceService_SyncSlowQueries_FullMethodName:
		ok, err := in.iamManager.CheckPermission(ctx, p, user)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check permission for method %q, err: %v", fullMethod, err)
		}
		if !ok {
			return status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q", fullMethod, p)
		}
	case
		v1pb.DatabaseService_GetDatabase_FullMethodName:
		projectIDs, err := in.getProjectIDsForDatabaseService(ctx, req)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check permission, err %v", err)
		}
		ok, err = in.iamManager.CheckPermission(ctx, p, user, projectIDs...)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check permission for method %q, err: %v", fullMethod, err)
		}
		if !ok {
			return status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q", fullMethod, p)
		}
	}

	return nil
}

func getDatabaseMessage(ctx context.Context, s *store.Store, databaseResourceName string) (*store.DatabaseMessage, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(databaseResourceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %q", databaseResourceName)
	}
	find := &store.FindDatabaseMessage{
		ShowDeleted: true,
	}
	databaseUID, isNumber := isNumber(databaseName)
	if isNumber {
		// Expected format: "instances/{ignored_value}/database/{uid}"
		find.UID = &databaseUID
	} else {
		// Expected format: "instances/{instance}/database/{database}"
		find.InstanceID = &instanceID
		find.DatabaseName = &databaseName
		instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
		}
		if instance == nil {
			return nil, errors.Wrapf(err, "instance not found")
		}
		find.IgnoreCaseSensitive = store.IgnoreDatabaseAndTableCaseSensitive(instance)
	}
	database, err := s.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Wrapf(err, "database %q not found", databaseResourceName)
	}
	return database, nil
}

func (in *ACLInterceptor) getProjectIDsForDatabaseService(ctx context.Context, req any) ([]string, error) {
	var projectIDs []string

	var databaseNames []string
	switch r := req.(type) {
	case *v1pb.GetDatabaseRequest:
		databaseNames = append(databaseNames, r.GetName())
	case *v1pb.SyncDatabaseRequest:
		databaseNames = append(databaseNames, r.GetName())
	case *v1pb.GetDatabaseMetadataRequest:
		databaseNames = append(databaseNames, r.GetName())
	case *v1pb.UpdateDatabaseMetadataRequest:
		databaseName, err := common.TrimSuffix(r.GetDatabaseMetadata().GetName(), "/metadata")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetDatabaseMetadata().GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.UpdateDatabaseRequest:
		databaseNames = append(databaseNames, r.GetDatabase().GetName())
		if hasPath(r.GetUpdateMask(), "project") {
			projectID, err := common.GetProjectID(r.GetDatabase().GetProject())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get projectID from %q", r.GetDatabase().GetProject())
			}
			projectIDs = append(projectIDs, projectID)
		}
	}

	for _, databaseName := range databaseNames {
		database, err := getDatabaseMessage(ctx, in.store, databaseName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %q", databaseName)
		}
		projectIDs = append(projectIDs, database.ProjectID)
	}

	return projectIDs, nil
}
