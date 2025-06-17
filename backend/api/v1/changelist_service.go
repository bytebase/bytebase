package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/proto/generated-go/v1/v1connect"
)

// ChangelistService implements the changelist service.
type ChangelistService struct {
	v1connect.UnimplementedChangelistServiceHandler
	store      *store.Store
	profile    *config.Profile
	iamManager *iam.Manager
}

// NewChangelistService creates a new ChangelistService.
func NewChangelistService(store *store.Store, profile *config.Profile, iamManager *iam.Manager) *ChangelistService {
	return &ChangelistService{
		store:      store,
		profile:    profile,
		iamManager: iamManager,
	}
}

// CreateChangelist creates a changelist.
func (s *ChangelistService) CreateChangelist(ctx context.Context, req *connect.Request[v1pb.CreateChangelistRequest]) (*connect.Response[v1pb.Changelist], error) {
	if req.Msg.Changelist == nil {
		return nil, status.Errorf(codes.InvalidArgument, "changelist must be set")
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}

	projectResourceID, err := common.GetProjectID(req.Msg.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project with resource id %q, err: %v", projectResourceID, err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project with resource id %q not found", projectResourceID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project with resource id %q had deleted", projectResourceID)
	}

	changelist, err := s.store.GetChangelist(ctx, &store.FindChangelistMessage{ProjectID: &project.ResourceID, ResourceID: &req.Msg.ChangelistId})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if changelist != nil {
		return nil, status.Errorf(codes.AlreadyExists, "changelist %q already exists", req.Msg.ChangelistId)
	}

	changelist, err = s.store.CreateChangelist(ctx, &store.ChangelistMessage{
		ProjectID:  project.ResourceID,
		ResourceID: req.Msg.ChangelistId,
		Payload:    convertV1ChangelistPayload(req.Msg.Changelist),
		CreatorID:  principalID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	v1Changelist, err := s.convertStoreChangelist(ctx, changelist)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(v1Changelist), nil
}

// GetChangelist gets a changelist.
func (s *ChangelistService) GetChangelist(ctx context.Context, req *connect.Request[v1pb.GetChangelistRequest]) (*connect.Response[v1pb.Changelist], error) {
	projectID, changelistID, err := common.GetProjectIDChangelistID(req.Msg.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}

	changelist, err := s.store.GetChangelist(ctx, &store.FindChangelistMessage{ProjectID: &project.ResourceID, ResourceID: &changelistID})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if changelist == nil {
		return nil, status.Errorf(codes.NotFound, "changelist %q not found", changelistID)
	}
	v1Changelist, err := s.convertStoreChangelist(ctx, changelist)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(v1Changelist), nil
}

// GetChangelist gets a changelist.
func (s *ChangelistService) ListChangelists(ctx context.Context, req *connect.Request[v1pb.ListChangelistsRequest]) (*connect.Response[v1pb.ListChangelistsResponse], error) {
	projectResourceID, err := common.GetProjectID(req.Msg.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project with resource id %q, err: %s", projectResourceID, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project with resource id %q not found", projectResourceID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project with resource id %q had deleted", projectResourceID)
	}

	changelists, err := s.store.ListChangelists(ctx, &store.FindChangelistMessage{
		ProjectID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &v1pb.ListChangelistsResponse{}
	for _, changelist := range changelists {
		v1Changelist, err := s.convertStoreChangelist(ctx, changelist)
		if err != nil {
			return nil, err
		}
		resp.Changelists = append(resp.Changelists, v1Changelist)
	}
	return connect.NewResponse(resp), nil
}

// UpdateChangelist updates a changelist.
func (s *ChangelistService) UpdateChangelist(ctx context.Context, req *connect.Request[v1pb.UpdateChangelistRequest]) (*connect.Response[v1pb.Changelist], error) {
	if req.Msg.Changelist == nil {
		return nil, status.Errorf(codes.InvalidArgument, "changelist must be set")
	}
	if req.Msg.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	projectID, changelistID, err := common.GetProjectIDChangelistID(req.Msg.Changelist.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}
	changelist, err := s.store.GetChangelist(ctx, &store.FindChangelistMessage{ProjectID: &project.ResourceID, ResourceID: &changelistID})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if changelist == nil {
		return nil, status.Errorf(codes.NotFound, "changelist %q not found", changelistID)
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	update := &store.UpdateChangelistMessage{
		UpdaterID:  user.ID,
		ProjectID:  project.ResourceID,
		ResourceID: changelistID,
	}
	newChangelist := convertV1ChangelistPayload(req.Msg.Changelist)

	for _, path := range req.Msg.UpdateMask.Paths {
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
		return nil, status.Error(codes.Internal, err.Error())
	}
	changelist, err = s.store.GetChangelist(ctx, &store.FindChangelistMessage{ProjectID: &project.ResourceID, ResourceID: &changelistID})
	if err != nil {
		return nil, err
	}

	v1Changelist, err := s.convertStoreChangelist(ctx, changelist)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(v1Changelist), nil
}

// DeleteChangelist deletes a changelist.
func (s *ChangelistService) DeleteChangelist(ctx context.Context, req *connect.Request[v1pb.DeleteChangelistRequest]) (*connect.Response[emptypb.Empty], error) {
	projectID, changelistID, err := common.GetProjectIDChangelistID(req.Msg.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}
	changelist, err := s.store.GetChangelist(ctx, &store.FindChangelistMessage{ProjectID: &project.ResourceID, ResourceID: &changelistID})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if changelist == nil {
		return nil, status.Errorf(codes.NotFound, "changelist %q not found", changelistID)
	}

	if err := s.store.DeleteChangelist(ctx, project.ResourceID, changelistID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
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
		return nil, status.Errorf(codes.Internal, "failed to get creator: %v", err)
	}
	if creator == nil {
		return nil, status.Errorf(codes.NotFound, "cannot find the creator: %d", changelist.CreatorID)
	}

	v1Changelist := &v1pb.Changelist{
		Name:        fmt.Sprintf("projects/%s/changelists/%s", changelist.ProjectID, changelist.ResourceID),
		Description: changelist.Payload.Description,
		UpdateTime:  timestamppb.New(changelist.UpdatedAt),
		Creator:     fmt.Sprintf("users/%s", creator.Email),
	}
	for _, change := range changelist.Payload.Changes {
		v1Changelist.Changes = append(v1Changelist.Changes, &v1pb.Changelist_Change{
			Sheet:  change.Sheet,
			Source: change.Source,
		})
	}
	return v1Changelist, nil
}
