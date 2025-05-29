package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// DatabaseGroupService implements the database group service.
type DatabaseGroupService struct {
	v1pb.UnimplementedDatabaseGroupServiceServer
	store          *store.Store
	profile        *config.Profile
	iamManager     *iam.Manager
	licenseService enterprise.LicenseService
}

// NewDatabaseGroupService creates a new ChangelistService.
func NewDatabaseGroupService(store *store.Store, profile *config.Profile, iamManager *iam.Manager, licenseService enterprise.LicenseService) *DatabaseGroupService {
	return &DatabaseGroupService{
		store:          store,
		profile:        profile,
		iamManager:     iamManager,
		licenseService: licenseService,
	}
}

// CreateDatabaseGroup creates a database group.
func (s *DatabaseGroupService) CreateDatabaseGroup(ctx context.Context, request *v1pb.CreateDatabaseGroupRequest) (*v1pb.DatabaseGroup, error) {
	if err := s.licenseService.IsFeatureEnabled(base.FeatureDatabaseGrouping); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	projectResourceID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", request.Parent)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", request.Parent)
	}

	if !isValidResourceID(request.DatabaseGroupId) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid database group id %q", request.DatabaseGroupId)
	}
	if request.DatabaseGroup.Title == "" {
		return nil, status.Errorf(codes.InvalidArgument, "database group database placeholder is required")
	}
	if request.DatabaseGroup.DatabaseExpr == nil || request.DatabaseGroup.DatabaseExpr.Expression == "" {
		return nil, status.Errorf(codes.InvalidArgument, "database group database expression is required")
	}
	if _, err := common.ValidateGroupCELExpr(request.DatabaseGroup.DatabaseExpr.Expression); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid database group expression: %v", err)
	}

	storeDatabaseGroup := &store.DatabaseGroupMessage{
		ResourceID:  request.DatabaseGroupId,
		ProjectID:   project.ResourceID,
		Placeholder: request.DatabaseGroup.Title,
		Expression:  request.DatabaseGroup.DatabaseExpr,
	}
	if request.ValidateOnly {
		return s.convertStoreToAPIDatabaseGroupFull(ctx, storeDatabaseGroup, projectResourceID)
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	hasPermission, err := s.iamManager.CheckPermission(ctx, iam.PermissionProjectsUpdate, user, project.ResourceID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission with error: %v", err.Error())
	}
	if !hasPermission {
		return nil, status.Errorf(codes.PermissionDenied, "user does not have permission %q", iam.PermissionProjectsUpdate)
	}

	databaseGroup, err := s.store.CreateDatabaseGroup(ctx, storeDatabaseGroup)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return s.convertStoreToAPIDatabaseGroupFull(ctx, databaseGroup, projectResourceID)
}

// UpdateDatabaseGroup updates a database group.
func (s *DatabaseGroupService) UpdateDatabaseGroup(ctx context.Context, request *v1pb.UpdateDatabaseGroupRequest) (*v1pb.DatabaseGroup, error) {
	if err := s.licenseService.IsFeatureEnabled(base.FeatureDatabaseGrouping); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	projectResourceID, databaseGroupResourceID, err := common.GetProjectIDDatabaseGroupID(request.DatabaseGroup.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectResourceID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", projectResourceID)
	}
	existedDatabaseGroup, err := s.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
		ProjectID:  &project.ResourceID,
		ResourceID: &databaseGroupResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if existedDatabaseGroup == nil {
		return nil, status.Errorf(codes.NotFound, "database group %q not found", databaseGroupResourceID)
	}

	var updateDatabaseGroup store.UpdateDatabaseGroupMessage
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			if request.DatabaseGroup.Title == "" {
				return nil, status.Errorf(codes.InvalidArgument, "database group database placeholder is required")
			}
			updateDatabaseGroup.Placeholder = &request.DatabaseGroup.Title
		case "database_expr":
			if request.DatabaseGroup.DatabaseExpr == nil || request.DatabaseGroup.DatabaseExpr.Expression == "" {
				return nil, status.Errorf(codes.InvalidArgument, "database group expr is required")
			}
			if _, err := common.ValidateGroupCELExpr(request.DatabaseGroup.DatabaseExpr.Expression); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid database group expression: %v", err)
			}
			updateDatabaseGroup.Expression = request.DatabaseGroup.DatabaseExpr
		default:
			return nil, status.Errorf(codes.InvalidArgument, "unsupported path: %q", path)
		}
	}
	databaseGroup, err := s.store.UpdateDatabaseGroup(ctx, existedDatabaseGroup.ProjectID, existedDatabaseGroup.ResourceID, &updateDatabaseGroup)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return s.convertStoreToAPIDatabaseGroupFull(ctx, databaseGroup, projectResourceID)
}

// DeleteDatabaseGroup deletes a database group.
func (s *DatabaseGroupService) DeleteDatabaseGroup(ctx context.Context, request *v1pb.DeleteDatabaseGroupRequest) (*emptypb.Empty, error) {
	projectResourceID, databaseGroupResourceID, err := common.GetProjectIDDatabaseGroupID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectResourceID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", projectResourceID)
	}
	existedDatabaseGroup, err := s.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
		ProjectID:  &project.ResourceID,
		ResourceID: &databaseGroupResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if existedDatabaseGroup == nil {
		return nil, status.Errorf(codes.NotFound, "database group %q not found", databaseGroupResourceID)
	}

	err = s.store.DeleteDatabaseGroup(ctx, existedDatabaseGroup.ProjectID, existedDatabaseGroup.ResourceID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// ListDatabaseGroups lists database groups.
func (s *DatabaseGroupService) ListDatabaseGroups(ctx context.Context, request *v1pb.ListDatabaseGroupsRequest) (*v1pb.ListDatabaseGroupsResponse, error) {
	projectResourceID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectResourceID)
	}

	databaseGroups, err := s.store.ListDatabaseGroups(ctx, &store.FindDatabaseGroupMessage{
		ProjectID: &project.ResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list database groups, err: %v", err)
	}

	var apiDatabaseGroups []*v1pb.DatabaseGroup
	for _, databaseGroup := range databaseGroups {
		if request.View == v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_BASIC || request.View == v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_UNSPECIFIED {
			apiDatabaseGroups = append(apiDatabaseGroups, convertStoreToAPIDatabaseGroupBasic(databaseGroup, project.ResourceID))
		} else {
			fullDatabaseGroup, err := s.convertStoreToAPIDatabaseGroupFull(ctx, databaseGroup, projectResourceID)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to convert database group %q to full view, err: %v", databaseGroup.ResourceID, err)
			}
			apiDatabaseGroups = append(apiDatabaseGroups, fullDatabaseGroup)
		}
	}
	return &v1pb.ListDatabaseGroupsResponse{
		DatabaseGroups: apiDatabaseGroups,
	}, nil
}

// GetDatabaseGroup gets a database group.
func (s *DatabaseGroupService) GetDatabaseGroup(ctx context.Context, request *v1pb.GetDatabaseGroupRequest) (*v1pb.DatabaseGroup, error) {
	projectResourceID, databaseGroupResourceID, err := common.GetProjectIDDatabaseGroupID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectResourceID)
	}
	databaseGroup, err := s.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
		ProjectID:  &project.ResourceID,
		ResourceID: &databaseGroupResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if databaseGroup == nil {
		return nil, status.Errorf(codes.NotFound, "database group %q not found", databaseGroupResourceID)
	}
	if request.View == v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_BASIC || request.View == v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_UNSPECIFIED {
		return convertStoreToAPIDatabaseGroupBasic(databaseGroup, projectResourceID), nil
	}
	return s.convertStoreToAPIDatabaseGroupFull(ctx, databaseGroup, projectResourceID)
}

func (s *DatabaseGroupService) convertStoreToAPIDatabaseGroupFull(ctx context.Context, databaseGroup *store.DatabaseGroupMessage, projectResourceID string) (*v1pb.DatabaseGroup, error) {
	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{
		ProjectID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	ret := convertStoreToAPIDatabaseGroupBasic(databaseGroup, projectResourceID)
	matches, unmatches, err := utils.GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx, databaseGroup, databases)
	if err != nil {
		return nil, err
	}
	for _, database := range matches {
		ret.MatchedDatabases = append(ret.MatchedDatabases, &v1pb.DatabaseGroup_Database{
			Name: fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		})
	}
	for _, database := range unmatches {
		ret.UnmatchedDatabases = append(ret.UnmatchedDatabases, &v1pb.DatabaseGroup_Database{
			Name: fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		})
	}
	return ret, nil
}

func convertStoreToAPIDatabaseGroupBasic(databaseGroup *store.DatabaseGroupMessage, projectResourceID string) *v1pb.DatabaseGroup {
	databaseGroupV1 := &v1pb.DatabaseGroup{
		Name:         fmt.Sprintf("%s/%s%s", common.FormatProject(projectResourceID), common.DatabaseGroupNamePrefix, databaseGroup.ResourceID),
		Title:        databaseGroup.Placeholder,
		DatabaseExpr: databaseGroup.Expression,
	}
	return databaseGroupV1
}
