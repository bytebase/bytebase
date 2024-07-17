package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// WorkspaceService implements the workspace service.
type WorkspaceService struct {
	v1pb.UnimplementedWorkspaceServiceServer
	store      *store.Store
	iamManager *iam.Manager
}

// NewWorkspaceService creates a new WorkspaceService.
func NewWorkspaceService(store *store.Store, iamManager *iam.Manager) *WorkspaceService {
	return &WorkspaceService{
		store:      store,
		iamManager: iamManager,
	}
}

func (s *WorkspaceService) GetIamPolicy(ctx context.Context, _ *v1pb.GetIamPolicyRequest) (*v1pb.IamPolicy, error) {
	policy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find iam policy with error: %v", err.Error())
	}

	return convertToV1IamPolicy(ctx, s.store, policy)
}

func (s *WorkspaceService) SetIamPolicy(ctx context.Context, request *v1pb.SetIamPolicyRequest) (*v1pb.IamPolicy, error) {
	userUID, err := s.checkUserIsAdmin(ctx)
	if err != nil {
		return nil, err
	}

	iamPolicy, err := convertToStoreIamPolicy(ctx, s.store, request.Policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert iam policy with error: %v", err.Error())
	}

	payloadBytes, err := protojson.Marshal(iamPolicy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal iam policy with error: %v", err.Error())
	}

	payloadStr := string(payloadBytes)
	patch := &store.UpdatePolicyMessage{
		UpdaterID:    userUID,
		ResourceType: api.PolicyResourceTypeWorkspace,
		Type:         api.PolicyTypeIAM,
		Payload:      &payloadStr,
	}

	updatedPolicy, err := s.store.UpdatePolicyV2(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	p := &storepb.IamPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(updatedPolicy.Payload), p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal iam policy payload")
	}
	return convertToV1IamPolicy(ctx, s.store, p)
}

func (s *WorkspaceService) PatchIamPolicy(ctx context.Context, request *v1pb.PatchIamPolicyRequest) (*v1pb.IamPolicy, error) {
	userUID, err := s.checkUserIsAdmin(ctx)
	if err != nil {
		return nil, err
	}

	// TODO(ed): Check if the user is the only workspace admin.
	storeMember, err := convertToStoreIamPolicyMember(ctx, s.store, request.Member)
	if err != nil {
		return nil, err
	}

	policy, err := s.store.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Member: storeMember,
		// TODO(ed): check roles.
		Roles:      request.Roles,
		UpdaterUID: userUID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to patch iam policy with error: %v", err.Error())
	}

	s.iamManager.RemoveCache(userUID)

	return convertToV1IamPolicy(ctx, s.store, policy)
}

func (s *WorkspaceService) checkUserIsAdmin(ctx context.Context) (int, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return 0, status.Errorf(codes.Internal, "failed to get user")
	}
	isAdmin, err := s.iamManager.CheckUserContainsWorkspaceRoles(ctx, user, api.WorkspaceAdmin)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "failed to get check admin role")
	}
	if !isAdmin {
		return 0, status.Errorf(codes.PermissionDenied, "only admin can set workspace iam policy")
	}
	return user.ID, nil
}
