package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ChangelistService implements the changelist service.
type ChangelistService struct {
	v1pb.UnimplementedChangelistServiceServer
	store *store.Store
}

// NewChangelistService creates a new ChangelistService.
func NewChangelistService(store *store.Store) *ChangelistService {
	return &ChangelistService{
		store: store,
	}
}

// CreateChangelist creates a changelist.
func (s *ChangelistService) CreateChangelist(ctx context.Context, request *v1pb.CreateChangelistRequest) (*v1pb.Changelist, error) {
	if request.Changelist == nil {
		return nil, status.Errorf(codes.InvalidArgument, "changelist must be set")
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}

	projectResourceID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project with resource id %q, err: %s", projectResourceID, err.Error()))
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q not found", projectResourceID))
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
	}

	changelist, err := s.store.GetChangelist(ctx, &store.FindChangelistMessage{ProjectID: &project.ResourceID, ResourceID: &request.ChangelistId})
	if err != nil {
		return nil, err
	}
	if changelist != nil {
		return nil, status.Errorf(codes.AlreadyExists, "changelist %q already exists", request.ChangelistId)
	}

	changelist, err = s.store.CreateChangelist(ctx, &store.ChangelistMessage{
		ProjectID:  project.ResourceID,
		ResourceID: request.ChangelistId,
		Payload:    convertV1ChangelistPayload(request.Changelist),
		CreatorID:  principalID,
	})
	if err != nil {
		return nil, err
	}
	v1Changelist, err := s.convertStoreChangelist(ctx, changelist)
	if err != nil {
		return nil, err
	}
	return v1Changelist, nil
}

// GetChangelist gets a changelist.
func (s *ChangelistService) GetChangelist(ctx context.Context, request *v1pb.GetChangelistRequest) (*v1pb.Changelist, error) {
	projectID, changelistID, err := common.GetProjectIDChangelistID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}

	changelist, err := s.store.GetChangelist(ctx, &store.FindChangelistMessage{ProjectID: &project.ResourceID, ResourceID: &changelistID})
	if err != nil {
		return nil, err
	}
	if changelist == nil {
		return nil, status.Errorf(codes.NotFound, "changelist %q not found", changelistID)
	}
	v1Changelist, err := s.convertStoreChangelist(ctx, changelist)
	if err != nil {
		return nil, err
	}
	return v1Changelist, nil
}

// GetChangelist gets a changelist.
func (s *ChangelistService) ListChangelists(ctx context.Context, request *v1pb.ListChangelistsRequest) (*v1pb.ListChangelistsResponse, error) {
	projectResourceID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	find := &store.FindChangelistMessage{}
	if projectResourceID != "-" {
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &projectResourceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project with resource id %q, err: %s", projectResourceID, err.Error()))
		}
		if project == nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q not found", projectResourceID))
		}
		if project.Deleted {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
		}
		find.ProjectID = &projectResourceID
	}
	changelists, err := s.store.ListChangelists(ctx, find)
	if err != nil {
		return nil, err
	}

	resp := &v1pb.ListChangelistsResponse{}
	for _, changelist := range changelists {
		v1Changelist, err := s.convertStoreChangelist(ctx, changelist)
		if err != nil {
			return nil, err
		}
		resp.Changelists = append(resp.Changelists, v1Changelist)
	}
	return resp, nil
}

// UpdateChangelist updates a changelist.
func (s *ChangelistService) UpdateChangelist(ctx context.Context, request *v1pb.UpdateChangelistRequest) (*v1pb.Changelist, error) {
	if request.Changelist == nil {
		return nil, status.Errorf(codes.InvalidArgument, "changelist must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	projectID, changelistID, err := common.GetProjectIDChangelistID(request.Changelist.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}
	changelist, err := s.store.GetChangelist(ctx, &store.FindChangelistMessage{ProjectID: &project.ResourceID, ResourceID: &changelistID})
	if err != nil {
		return nil, err
	}
	if changelist == nil {
		return nil, status.Errorf(codes.NotFound, "changelist %q not found", changelistID)
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	update := &store.UpdateChangelistMessage{
		UpdaterID:  principalID,
		ProjectID:  project.ResourceID,
		ResourceID: changelistID,
	}
	newChangelist := convertV1ChangelistPayload(request.Changelist)

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "description":
			changelist.Payload.Description = newChangelist.Description
		case "changes":
			changelist.Payload.Changes = newChangelist.Changes
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupport update_mask "%s"`, path)
		}
	}
	update.Payload = newChangelist
	if err := s.store.UpdateChangelist(ctx, update); err != nil {
		return nil, err
	}
	changelist, err = s.store.GetChangelist(ctx, &store.FindChangelistMessage{ProjectID: &project.ResourceID, ResourceID: &changelistID})
	if err != nil {
		return nil, err
	}

	v1Changelist, err := s.convertStoreChangelist(ctx, changelist)
	if err != nil {
		return nil, err
	}
	return v1Changelist, nil
}

// DeleteChangelist deletes a changelist.
func (s *ChangelistService) DeleteChangelist(ctx context.Context, request *v1pb.DeleteChangelistRequest) (*emptypb.Empty, error) {
	projectID, changelistID, err := common.GetProjectIDChangelistID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}
	changelist, err := s.store.GetChangelist(ctx, &store.FindChangelistMessage{ProjectID: &project.ResourceID, ResourceID: &changelistID})
	if err != nil {
		return nil, err
	}
	if changelist == nil {
		return nil, status.Errorf(codes.NotFound, "changelist %q not found", changelistID)
	}

	if err := s.store.DeleteChangelist(ctx, project.ResourceID, changelistID); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func convertV1ChangelistPayload(changelist *v1pb.Changelist) *storepb.Changelist {
	storeChangelist := &storepb.Changelist{
		Description: changelist.Description,
	}
	for _, change := range changelist.Changes {
		storeChangelist.Changes = append(storeChangelist.Changes, &storepb.Changelist_Change{
			Sheet:  change.Sheet,
			Source: change.Source,
		})
	}
	return storeChangelist
}

func (s *ChangelistService) convertStoreChangelist(ctx context.Context, changelist *store.ChangelistMessage) (*v1pb.Changelist, error) {
	creator, err := s.store.GetUserByID(ctx, changelist.CreatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get creator: %v", err))
	}
	if creator == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("cannot find the creator: %d", changelist.CreatorID))
	}
	updater, err := s.store.GetUserByID(ctx, changelist.UpdaterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get updater: %v", err))
	}
	if updater == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("cannot find the updater: %d", changelist.UpdaterID))
	}

	v1Changelist := &v1pb.Changelist{
		Name:        fmt.Sprintf("projects/%s/changelists/%s", changelist.ProjectID, changelist.ResourceID),
		Description: changelist.Payload.Description,
		CreateTime:  timestamppb.New(changelist.CreatedTime),
		UpdateTime:  timestamppb.New(changelist.UpdatedTime),
		Creator:     fmt.Sprintf("users/%s", creator.Email),
		Updater:     fmt.Sprintf("users/%s", updater.Email),
	}
	for _, change := range changelist.Payload.Changes {
		v1Changelist.Changes = append(v1Changelist.Changes, &v1pb.Changelist_Change{
			Sheet:  change.Sheet,
			Source: change.Source,
		})
	}
	return v1Changelist, nil
}
