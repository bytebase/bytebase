package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/enterprise"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// DatabaseGroupService implements the database group service.
type DatabaseGroupService struct {
	v1connect.UnimplementedDatabaseGroupServiceHandler
	store          *store.Store
	licenseService *enterprise.LicenseService
}

// NewDatabaseGroupService creates a new DatabaseGroupService.
func NewDatabaseGroupService(store *store.Store, licenseService *enterprise.LicenseService) *DatabaseGroupService {
	return &DatabaseGroupService{
		store:          store,
		licenseService: licenseService,
	}
}

// CreateDatabaseGroup creates a database group.
func (s *DatabaseGroupService) CreateDatabaseGroup(ctx context.Context, req *connect.Request[v1pb.CreateDatabaseGroupRequest]) (*connect.Response[v1pb.DatabaseGroup], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DATABASE_GROUPS); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	projectResourceID, err := common.GetProjectID(req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", req.Msg.Parent))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has been deleted", req.Msg.Parent))
	}

	if !isValidResourceID(req.Msg.DatabaseGroupId) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid database group id %q", req.Msg.DatabaseGroupId))
	}
	if req.Msg.DatabaseGroup.Title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("database group database placeholder is required"))
	}
	if req.Msg.DatabaseGroup.DatabaseExpr == nil || req.Msg.DatabaseGroup.DatabaseExpr.Expression == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("database group database expression is required"))
	}
	if _, err := common.ValidateGroupCELExpr(req.Msg.DatabaseGroup.DatabaseExpr.Expression); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid database group expression"))
	}

	storeDatabaseGroup := &store.DatabaseGroupMessage{
		ResourceID: req.Msg.DatabaseGroupId,
		ProjectID:  project.ResourceID,
		Title:      req.Msg.DatabaseGroup.Title,
		Expression: req.Msg.DatabaseGroup.DatabaseExpr,
	}
	if req.Msg.ValidateOnly {
		result, err := convertStoreToV1DatabaseGroupWithView(ctx, s.store, storeDatabaseGroup, projectResourceID, v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(result), nil
	}

	databaseGroup, err := s.store.CreateDatabaseGroup(ctx, storeDatabaseGroup)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result, err := convertStoreToV1DatabaseGroupWithView(ctx, s.store, databaseGroup, projectResourceID, v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

// UpdateDatabaseGroup updates a database group.
func (s *DatabaseGroupService) UpdateDatabaseGroup(ctx context.Context, req *connect.Request[v1pb.UpdateDatabaseGroupRequest]) (*connect.Response[v1pb.DatabaseGroup], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DATABASE_GROUPS); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	projectResourceID, databaseGroupResourceID, err := common.GetProjectIDDatabaseGroupID(req.Msg.DatabaseGroup.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectResourceID))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has been deleted", projectResourceID))
	}

	existedDatabaseGroup, err := s.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
		ProjectID:  &project.ResourceID,
		ResourceID: &databaseGroupResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if existedDatabaseGroup == nil {
		if req.Msg.AllowMissing {
			return s.CreateDatabaseGroup(ctx, connect.NewRequest(&v1pb.CreateDatabaseGroupRequest{
				Parent:          common.FormatProject(project.ResourceID),
				DatabaseGroupId: databaseGroupResourceID,
				DatabaseGroup:   req.Msg.DatabaseGroup,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database group %q not found", databaseGroupResourceID))
	}

	var updateDatabaseGroup store.UpdateDatabaseGroupMessage
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "title":
			if req.Msg.DatabaseGroup.Title == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("database group database placeholder is required"))
			}
			updateDatabaseGroup.Title = &req.Msg.DatabaseGroup.Title
		case "database_expr":
			if req.Msg.DatabaseGroup.DatabaseExpr == nil || req.Msg.DatabaseGroup.DatabaseExpr.Expression == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("database group expr is required"))
			}
			if _, err := common.ValidateGroupCELExpr(req.Msg.DatabaseGroup.DatabaseExpr.Expression); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid database group expression"))
			}
			updateDatabaseGroup.Expression = req.Msg.DatabaseGroup.DatabaseExpr
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported path: %q", path))
		}
	}
	databaseGroup, err := s.store.UpdateDatabaseGroup(ctx, existedDatabaseGroup.ProjectID, existedDatabaseGroup.ResourceID, &updateDatabaseGroup)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result, err := convertStoreToV1DatabaseGroupWithView(ctx, s.store, databaseGroup, projectResourceID, v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

// DeleteDatabaseGroup deletes a database group.
func (s *DatabaseGroupService) DeleteDatabaseGroup(ctx context.Context, req *connect.Request[v1pb.DeleteDatabaseGroupRequest]) (*connect.Response[emptypb.Empty], error) {
	projectResourceID, databaseGroupResourceID, err := common.GetProjectIDDatabaseGroupID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectResourceID))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has been deleted", projectResourceID))
	}
	existedDatabaseGroup, err := s.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
		ProjectID:  &project.ResourceID,
		ResourceID: &databaseGroupResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if existedDatabaseGroup == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database group %q not found", databaseGroupResourceID))
	}

	err = s.store.DeleteDatabaseGroup(ctx, existedDatabaseGroup.ProjectID, existedDatabaseGroup.ResourceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ListDatabaseGroups lists database groups.
func (s *DatabaseGroupService) ListDatabaseGroups(ctx context.Context, req *connect.Request[v1pb.ListDatabaseGroupsRequest]) (*connect.Response[v1pb.ListDatabaseGroupsResponse], error) {
	projectResourceID, err := common.GetProjectID(req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectResourceID))
	}

	databaseGroups, err := s.store.ListDatabaseGroups(ctx, &store.FindDatabaseGroupMessage{
		ProjectID: &project.ResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list database groups"))
	}

	var allProjectDatabases []*store.DatabaseMessage
	if req.Msg.View == v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL {
		allProjectDatabases, err = s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &projectResourceID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	var apiDatabaseGroups []*v1pb.DatabaseGroup
	for _, databaseGroup := range databaseGroups {
		group, err := convertStoreToV1DatabaseGroup(ctx, databaseGroup, projectResourceID, allProjectDatabases)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert database group %q", databaseGroup.ResourceID))
		}
		apiDatabaseGroups = append(apiDatabaseGroups, group)
	}
	return connect.NewResponse(&v1pb.ListDatabaseGroupsResponse{
		DatabaseGroups: apiDatabaseGroups,
	}), nil
}

// GetDatabaseGroup gets a database group.
func (s *DatabaseGroupService) GetDatabaseGroup(ctx context.Context, req *connect.Request[v1pb.GetDatabaseGroupRequest]) (*connect.Response[v1pb.DatabaseGroup], error) {
	result, err := getDatabaseGroupByName(ctx, s.store, req.Msg.Name, req.Msg.View)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func getDatabaseGroupByName(ctx context.Context, stores *store.Store, databaseGroupName string, view v1pb.DatabaseGroupView) (*v1pb.DatabaseGroup, error) {
	projectResourceID, databaseGroupResourceID, err := common.GetProjectIDDatabaseGroupID(databaseGroupName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := stores.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectResourceID))
	}
	databaseGroup, err := stores.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
		ProjectID:  &project.ResourceID,
		ResourceID: &databaseGroupResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if databaseGroup == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database group %q not found", databaseGroupResourceID))
	}
	return convertStoreToV1DatabaseGroupWithView(ctx, stores, databaseGroup, projectResourceID, view)
}

func convertStoreToV1DatabaseGroupWithView(ctx context.Context, stores *store.Store, databaseGroup *store.DatabaseGroupMessage, projectResourceID string, view v1pb.DatabaseGroupView) (*v1pb.DatabaseGroup, error) {
	var allProjectDatabases []*store.DatabaseMessage
	if view == v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL {
		databases, err := stores.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &projectResourceID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		allProjectDatabases = databases
	}

	return convertStoreToV1DatabaseGroup(ctx, databaseGroup, projectResourceID, allProjectDatabases)
}

func convertStoreToV1DatabaseGroup(
	ctx context.Context,
	databaseGroup *store.DatabaseGroupMessage,
	projectResourceID string,
	databases []*store.DatabaseMessage,
) (*v1pb.DatabaseGroup, error) {
	ret := &v1pb.DatabaseGroup{
		Name:         fmt.Sprintf("%s/%s%s", common.FormatProject(projectResourceID), common.DatabaseGroupNamePrefix, databaseGroup.ResourceID),
		Title:        databaseGroup.Title,
		DatabaseExpr: databaseGroup.Expression,
	}

	matches, err := utils.GetMatchedDatabasesInDatabaseGroup(ctx, databaseGroup, databases)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get matched databases for group %v with error: %v", ret.Name, err.Error()))
	}
	for _, database := range matches {
		ret.MatchedDatabases = append(ret.MatchedDatabases, &v1pb.DatabaseGroup_Database{
			Name: common.FormatDatabase(database.InstanceID, database.DatabaseName),
		})
	}
	return ret, nil
}
