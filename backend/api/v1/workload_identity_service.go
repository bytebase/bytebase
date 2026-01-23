package v1

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// WorkloadIdentityService implements the workload identity service.
type WorkloadIdentityService struct {
	v1connect.UnimplementedWorkloadIdentityServiceHandler
	store      *store.Store
	iamManager *iam.Manager
}

// NewWorkloadIdentityService creates a new WorkloadIdentityService.
func NewWorkloadIdentityService(store *store.Store, iamManager *iam.Manager) *WorkloadIdentityService {
	return &WorkloadIdentityService{
		store:      store,
		iamManager: iamManager,
	}
}

// CreateWorkloadIdentity creates a new workload identity.
func (s *WorkloadIdentityService) CreateWorkloadIdentity(ctx context.Context, request *connect.Request[v1pb.CreateWorkloadIdentityRequest]) (*connect.Response[v1pb.WorkloadIdentity], error) {
	// Parse parent to determine workspace vs project level
	var projectID *string
	parent := request.Msg.Parent
	if parent != "" {
		// project-level workload identity: parent = "projects/{project}"
		if strings.HasPrefix(parent, common.ProjectNamePrefix) {
			pid, err := common.GetProjectID(parent)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid parent %q", parent))
			}
			projectID = &pid
		} else {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid parent format %q, expected projects/{project} or empty", parent))
		}
	}

	workloadIdentityID := request.Msg.WorkloadIdentityId
	if workloadIdentityID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("workload_identity_id is required"))
	}

	wi := request.Msg.WorkloadIdentity
	if wi == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("workload_identity is required"))
	}

	// Build email using helper function
	projectIDStr := ""
	if projectID != nil {
		projectIDStr = *projectID
	}
	email := common.BuildWorkloadIdentityEmail(workloadIdentityID, projectIDStr)

	// Check for duplicate email
	existingWI, err := s.store.GetWorkloadIdentityByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check for existing workload identity"))
	}
	if existingWI != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.Errorf("workload identity with email %q already exists", email))
	}

	// Convert API workload identity config to store workload identity config
	var storeConfig *storepb.WorkloadIdentityConfig
	if wi.WorkloadIdentityConfig != nil {
		storeConfig = convertToStoreWorkloadIdentityConfig(wi.WorkloadIdentityConfig)
	}

	// Create the workload identity
	createdWI, err := s.store.CreateWorkloadIdentity(ctx, &store.CreateWorkloadIdentityMessage{
		Email:   email,
		Name:    wi.Title,
		Project: projectID,
		Config:  storeConfig,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create workload identity"))
	}

	return connect.NewResponse(convertToWorkloadIdentity(createdWI)), nil
}

// GetWorkloadIdentity gets a workload identity by name.
func (s *WorkloadIdentityService) GetWorkloadIdentity(ctx context.Context, request *connect.Request[v1pb.GetWorkloadIdentityRequest]) (*connect.Response[v1pb.WorkloadIdentity], error) {
	email, err := common.GetWorkloadIdentityEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	wi, err := s.store.GetWorkloadIdentityByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get workload identity"))
	}
	if wi == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("workload identity %q not found", email))
	}

	return connect.NewResponse(convertToWorkloadIdentity(wi)), nil
}

// BatchGetWorkloadIdentities gets workload identities in batch.
func (s *WorkloadIdentityService) BatchGetWorkloadIdentities(ctx context.Context, request *connect.Request[v1pb.BatchGetWorkloadIdentitiesRequest]) (*connect.Response[v1pb.BatchGetWorkloadIdentitiesResponse], error) {
	emails := make([]string, 0, len(request.Msg.Names))
	for _, name := range request.Msg.Names {
		email, err := common.GetWorkloadIdentityEmail(name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if !common.IsWorkloadIdentityEmail(email) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email %v is not workload identity", email))
		}
		emails = append(emails, email)
	}

	users, err := s.store.BatchGetUsersByEmails(ctx, emails)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to batch get workload identities"))
	}

	response := &v1pb.BatchGetWorkloadIdentitiesResponse{}
	for _, user := range users {
		response.WorkloadIdentities = append(response.WorkloadIdentities, convertUserToWorkloadIdentity(user))
	}

	return connect.NewResponse(response), nil
}

// ListWorkloadIdentities lists workload identities.
func (s *WorkloadIdentityService) ListWorkloadIdentities(ctx context.Context, request *connect.Request[v1pb.ListWorkloadIdentitiesRequest]) (*connect.Response[v1pb.ListWorkloadIdentitiesResponse], error) {
	// Parse parent to determine workspace vs project level
	var projectID *string
	parent := request.Msg.Parent
	if parent != "" {
		if strings.HasPrefix(parent, common.ProjectNamePrefix) {
			pid, err := common.GetProjectID(parent)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid parent %q", parent))
			}
			projectID = &pid
		} else {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid parent format %q, expected projects/{project} or empty", parent))
		}
	} else {
		// workspace-level list - use empty string to filter workspace-level WIs
		emptyProjectID := ""
		projectID = &emptyProjectID
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.Msg.PageToken,
		limit:   int(request.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	// List workload identities using the store method with project filtering
	wis, err := s.store.ListWorkloadIdentities(ctx, &store.FindWorkloadIdentityMessage{
		Project:     projectID,
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
		ShowDeleted: request.Msg.ShowDeleted,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list workload identities"))
	}

	nextPageToken := ""
	if len(wis) == limitPlusOne {
		wis = wis[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal next page token"))
		}
	}

	response := &v1pb.ListWorkloadIdentitiesResponse{
		NextPageToken: nextPageToken,
	}
	for _, wi := range wis {
		response.WorkloadIdentities = append(response.WorkloadIdentities, convertToWorkloadIdentity(wi))
	}

	return connect.NewResponse(response), nil
}

// UpdateWorkloadIdentity updates a workload identity.
func (s *WorkloadIdentityService) UpdateWorkloadIdentity(ctx context.Context, request *connect.Request[v1pb.UpdateWorkloadIdentityRequest]) (*connect.Response[v1pb.WorkloadIdentity], error) {
	if request.Msg.WorkloadIdentity == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("workload_identity is required"))
	}
	if request.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask is required"))
	}

	email, err := common.GetWorkloadIdentityEmail(request.Msg.WorkloadIdentity.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	wi, err := s.store.GetWorkloadIdentityByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get workload identity"))
	}
	if wi == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("workload identity %q not found", email))
	}
	if wi.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("workload identity %q has been deleted", email))
	}

	patch := &store.UpdateWorkloadIdentityMessage{}

	for _, path := range request.Msg.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Name = &request.Msg.WorkloadIdentity.Title
		case "workload_identity_config":
			if request.Msg.WorkloadIdentity.WorkloadIdentityConfig != nil {
				patch.Config = convertToStoreWorkloadIdentityConfig(request.Msg.WorkloadIdentity.WorkloadIdentityConfig)
			} else {
				patch.Config = &storepb.WorkloadIdentityConfig{}
			}
		default:
			// Ignore unknown fields
		}
	}

	updatedWI, err := s.store.UpdateWorkloadIdentity(ctx, wi, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update workload identity"))
	}

	return connect.NewResponse(convertToWorkloadIdentity(updatedWI)), nil
}

// DeleteWorkloadIdentity deletes a workload identity.
func (s *WorkloadIdentityService) DeleteWorkloadIdentity(ctx context.Context, request *connect.Request[v1pb.DeleteWorkloadIdentityRequest]) (*connect.Response[emptypb.Empty], error) {
	email, err := common.GetWorkloadIdentityEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	wi, err := s.store.GetWorkloadIdentityByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get workload identity"))
	}
	if wi == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("workload identity %q not found", email))
	}
	if wi.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("workload identity %q has been deleted", email))
	}

	// Soft delete
	if err := s.store.DeleteWorkloadIdentity(ctx, wi); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to delete workload identity"))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// UndeleteWorkloadIdentity restores a deleted workload identity.
func (s *WorkloadIdentityService) UndeleteWorkloadIdentity(ctx context.Context, request *connect.Request[v1pb.UndeleteWorkloadIdentityRequest]) (*connect.Response[v1pb.WorkloadIdentity], error) {
	email, err := common.GetWorkloadIdentityEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	wi, err := s.store.GetWorkloadIdentityByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get workload identity"))
	}
	if wi == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("workload identity %q not found", email))
	}
	if !wi.MemberDeleted {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("workload identity %q is already active", email))
	}

	// Restore
	restoredWI, err := s.store.UndeleteWorkloadIdentity(ctx, wi)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to undelete workload identity"))
	}

	return connect.NewResponse(convertToWorkloadIdentity(restoredWI)), nil
}

// convertToWorkloadIdentity converts a store.WorkloadIdentityMessage to a v1pb.WorkloadIdentity.
func convertToWorkloadIdentity(wi *store.WorkloadIdentityMessage) *v1pb.WorkloadIdentity {
	result := &v1pb.WorkloadIdentity{
		Name:  common.FormatWorkloadIdentityEmail(wi.Email),
		State: convertDeletedToState(wi.MemberDeleted),
		Email: wi.Email,
		Title: wi.Name,
	}

	// Convert workload identity config
	if wi.Config != nil {
		result.WorkloadIdentityConfig = convertToAPIWorkloadIdentityConfig(wi.Config)
	}

	return result
}

// convertUserToWorkloadIdentity converts a store.UserMessage to a v1pb.WorkloadIdentity.
func convertUserToWorkloadIdentity(user *store.UserMessage) *v1pb.WorkloadIdentity {
	result := &v1pb.WorkloadIdentity{
		Name:  common.FormatWorkloadIdentityEmail(user.Email),
		State: convertDeletedToState(user.MemberDeleted),
		Email: user.Email,
		Title: user.Name,
	}

	// Convert workload identity config from user profile
	if user.Profile != nil && user.Profile.WorkloadIdentityConfig != nil {
		result.WorkloadIdentityConfig = convertToAPIWorkloadIdentityConfig(user.Profile.WorkloadIdentityConfig)
	}

	return result
}

// convertToStoreWorkloadIdentityConfig converts API WorkloadIdentityConfig to store WorkloadIdentityConfig.
func convertToStoreWorkloadIdentityConfig(config *v1pb.WorkloadIdentityConfig) *storepb.WorkloadIdentityConfig {
	if config == nil {
		return nil
	}
	return &storepb.WorkloadIdentityConfig{
		ProviderType:     storepb.WorkloadIdentityConfig_ProviderType(config.ProviderType),
		IssuerUrl:        config.IssuerUrl,
		AllowedAudiences: config.AllowedAudiences,
		SubjectPattern:   config.SubjectPattern,
	}
}

// convertToAPIWorkloadIdentityConfig converts store WorkloadIdentityConfig to API WorkloadIdentityConfig.
func convertToAPIWorkloadIdentityConfig(config *storepb.WorkloadIdentityConfig) *v1pb.WorkloadIdentityConfig {
	if config == nil {
		return nil
	}
	return &v1pb.WorkloadIdentityConfig{
		ProviderType:     v1pb.WorkloadIdentityConfig_ProviderType(config.ProviderType),
		IssuerUrl:        config.IssuerUrl,
		AllowedAudiences: config.AllowedAudiences,
		SubjectPattern:   config.SubjectPattern,
	}
}
