package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	vcsplugin "github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/gitlab"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// VCSConnectorService implements the vcs connector service.
type VCSConnectorService struct {
	v1pb.UnimplementedVCSConnectorServiceServer
	store *store.Store
}

// NewVCSConnectorService creates a new VCSConnectorService.
func NewVCSConnectorService(store *store.Store) *VCSConnectorService {
	return &VCSConnectorService{
		store: store,
	}
}

// CreateVCSConnector creates a vcs connector.
func (s *VCSConnectorService) CreateVCSConnector(ctx context.Context, request *v1pb.CreateVCSConnectorRequest) (*v1pb.VCSConnector, error) {
	if request.VcsConnector == nil {
		return nil, status.Errorf(codes.InvalidArgument, "vcs connector must be set")
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace setting: %v", err)
	}
	if setting.ExternalUrl == "" {
		return nil, status.Errorf(codes.FailedPrecondition, setupExternalURLError)
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

	vcsConnector, err := s.store.GetVCSConnector(ctx, &store.FindVCSConnectorMessage{ProjectID: &project.ResourceID, ResourceID: &request.VcsConnectorId})
	if err != nil {
		return nil, err
	}
	if vcsConnector != nil {
		return nil, status.Errorf(codes.AlreadyExists, "vcs connector %q already exists", request.VcsConnectorId)
	}

	vcsResourceID, err := common.GetVCSProviderID(request.GetVcsConnector().GetVcsProvider())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	vcs, err := s.store.GetVCSProvider(ctx, &store.FindVCSProviderMessage{ResourceID: &vcsResourceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find vcs: %s", err.Error())
	}
	if vcs == nil {
		return nil, status.Errorf(codes.NotFound, "vcs %s not found", vcsResourceID)
	}

	// Check branch existence.
	notFound, err := isBranchNotFound(
		ctx,
		vcs,
		vcs.AccessToken,
		request.GetVcsConnector().ExternalId,
		request.GetVcsConnector().Branch,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check branch %s with error: %v", request.GetVcsConnector().Branch, err.Error())
	}
	if notFound {
		return nil, status.Errorf(codes.NotFound, "branch %s not found in repository %s", request.GetVcsConnector().Branch, request.GetVcsConnector().FullPath)
	}

	baseDir := request.GetVcsConnector().BaseDirectory
	// Azure DevOps base directory should start with /.
	if vcs.Type == vcsplugin.AzureDevOps {
		if !strings.HasPrefix(baseDir, "/") {
			baseDir = "/" + request.GetVcsConnector().BaseDirectory
		}
	} else {
		baseDir = strings.Trim(request.GetVcsConnector().BaseDirectory, "/")
	}

	workspaceID, err := s.store.GetWorkspaceID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace id with error: %v", err.Error())
	}
	secretToken, err := common.RandomString(gitlab.SecretTokenLength)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate random secret token for vcs with error: %v", err.Error())
	}
	vcsConnectorCreate := &store.VCSConnectorMessage{
		ProjectID:  project.ResourceID,
		ResourceID: request.VcsConnectorId,
		CreatorID:  principalID,

		VCSUID:        vcs.ID,
		VCSResourceID: vcs.ResourceID,
		Payload: &storepb.VCSConnector{
			Title:              request.GetVcsConnector().Title,
			FullPath:           request.GetVcsConnector().FullPath,
			WebUrl:             request.GetVcsConnector().WebUrl,
			Branch:             request.GetVcsConnector().Branch,
			BaseDirectory:      baseDir,
			ExternalId:         request.GetVcsConnector().ExternalId,
			WebhookSecretToken: secretToken,
		},
	}

	// Create the webhook.
	bytebaseEndpointURL := setting.GitopsWebhookUrl
	if bytebaseEndpointURL == "" {
		bytebaseEndpointURL = setting.ExternalUrl
	}
	webhookEndpointID := fmt.Sprintf("workspaces/%s/projects/%s/vcsConnectors/%s", workspaceID, project.ResourceID, request.VcsConnectorId)
	webhookID, err := createVCSWebhook(ctx, vcs.Type, webhookEndpointID, secretToken, vcs.AccessToken, vcs.InstanceURL, vcsConnectorCreate.Payload.ExternalId, bytebaseEndpointURL)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create webhook for project %s with error: %v", vcsConnectorCreate.ProjectID, err.Error())
	}
	vcsConnectorCreate.Payload.ExternalWebhookId = webhookID

	vcsConnector, err = s.store.CreateVCSConnector(ctx, vcsConnectorCreate)
	if err != nil {
		return nil, err
	}
	v1VCSConnector, err := s.convertStoreVCSConnector(ctx, vcsConnector)
	if err != nil {
		return nil, err
	}
	return v1VCSConnector, nil
}

// GetVCSConnector gets a vcs connector.
func (s *VCSConnectorService) GetVCSConnector(ctx context.Context, request *v1pb.GetVCSConnectorRequest) (*v1pb.VCSConnector, error) {
	projectID, vcsConnectorID, err := common.GetProjectVCSConnectorID(request.Name)
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

	vcsConnector, err := s.store.GetVCSConnector(ctx, &store.FindVCSConnectorMessage{ProjectID: &project.ResourceID, ResourceID: &vcsConnectorID})
	if err != nil {
		return nil, err
	}
	if vcsConnector == nil {
		return nil, status.Errorf(codes.NotFound, "vcs connector %q not found", vcsConnectorID)
	}
	v1VCSConnector, err := s.convertStoreVCSConnector(ctx, vcsConnector)
	if err != nil {
		return nil, err
	}
	return v1VCSConnector, nil
}

// GetVCSConnector gets a vcs connector.
func (s *VCSConnectorService) ListVCSConnectors(ctx context.Context, request *v1pb.ListVCSConnectorsRequest) (*v1pb.ListVCSConnectorsResponse, error) {
	projectID, err := common.GetProjectID(request.Parent)
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

	vcsConnectors, err := s.store.ListVCSConnectors(ctx, &store.FindVCSConnectorMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, err
	}

	resp := &v1pb.ListVCSConnectorsResponse{}
	for _, vcsConnector := range vcsConnectors {
		v1VCSConnector, err := s.convertStoreVCSConnector(ctx, vcsConnector)
		if err != nil {
			return nil, err
		}
		resp.VcsConnectors = append(resp.VcsConnectors, v1VCSConnector)
	}
	return resp, nil
}

// UpdateVCSConnector updates a vcs connector.
func (s *VCSConnectorService) UpdateVCSConnector(ctx context.Context, request *v1pb.UpdateVCSConnectorRequest) (*v1pb.VCSConnector, error) {
	if request.VcsConnector == nil {
		return nil, status.Errorf(codes.InvalidArgument, "vcs connector must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	projectID, vcsConnectorID, err := common.GetProjectVCSConnectorID(request.GetVcsConnector().GetName())
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

	vcsConnector, err := s.store.GetVCSConnector(ctx, &store.FindVCSConnectorMessage{ProjectID: &project.ResourceID, ResourceID: &vcsConnectorID})
	if err != nil {
		return nil, err
	}
	if vcsConnector == nil {
		return nil, status.Errorf(codes.NotFound, "vcs connector %q not found", vcsConnectorID)
	}

	vcs, err := s.store.GetVCSProvider(ctx, &store.FindVCSProviderMessage{ResourceID: &vcsConnector.VCSResourceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find vcs: %s", err.Error())
	}
	if vcs == nil {
		return nil, status.Errorf(codes.NotFound, "vcs %s not found", vcsConnector.VCSResourceID)
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	update := &store.UpdateVCSConnectorMessage{
		ProjectID: project.ResourceID,
		UpdaterID: user.ID,
		UID:       vcsConnector.UID,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "branch":
			update.Branch = &request.GetVcsConnector().Branch
		case "base_directory":
			baseDir := request.GetVcsConnector().BaseDirectory
			// Azure DevOps base directory should start with /.
			if vcs.Type == vcsplugin.AzureDevOps {
				if !strings.HasPrefix(baseDir, "/") {
					baseDir = "/" + request.GetVcsConnector().BaseDirectory
				}
			} else {
				baseDir = strings.Trim(request.GetVcsConnector().BaseDirectory, "/")
			}
			update.BaseDirectory = &baseDir
		}
	}

	// Check branch existence.
	if v := update.Branch; v != nil {
		if *v == "" {
			return nil, status.Errorf(codes.InvalidArgument, "branch must be specified")
		}
		notFound, err := isBranchNotFound(
			ctx,
			vcs,
			vcs.AccessToken,
			vcsConnector.Payload.ExternalId,
			*v,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check branch: %v", err.Error())
		}
		if notFound {
			return nil, status.Errorf(codes.NotFound, "branch %s not found in repository %s", *v, vcsConnector.Payload.FullPath)
		}
	}

	if err := s.store.UpdateVCSConnector(ctx, update); err != nil {
		return nil, err
	}
	vcsConnector, err = s.store.GetVCSConnector(ctx, &store.FindVCSConnectorMessage{ProjectID: &project.ResourceID, ResourceID: &vcsConnectorID})
	if err != nil {
		return nil, err
	}

	v1VCSConnector, err := s.convertStoreVCSConnector(ctx, vcsConnector)
	if err != nil {
		return nil, err
	}
	return v1VCSConnector, nil
}

// DeleteVCSConnector deletes a vcs connector.
func (s *VCSConnectorService) DeleteVCSConnector(ctx context.Context, request *v1pb.DeleteVCSConnectorRequest) (*emptypb.Empty, error) {
	projectID, vcsConnectorID, err := common.GetProjectVCSConnectorID(request.GetName())
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

	vcsConnector, err := s.store.GetVCSConnector(ctx, &store.FindVCSConnectorMessage{ProjectID: &project.ResourceID, ResourceID: &vcsConnectorID})
	if err != nil {
		return nil, err
	}
	if vcsConnector == nil {
		return nil, status.Errorf(codes.NotFound, "vcs connector %q not found", vcsConnectorID)
	}

	vcs, err := s.store.GetVCSProvider(ctx, &store.FindVCSProviderMessage{ResourceID: &vcsConnector.VCSResourceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find vcs: %s", err.Error())
	}
	if vcs == nil {
		return nil, status.Errorf(codes.NotFound, "vcs %d not found", vcsConnector.UID)
	}

	if err := s.store.DeleteVCSConnector(ctx, project.ResourceID, vcsConnectorID); err != nil {
		return nil, err
	}

	// Delete the webhook, and fail-open.
	if err = vcsplugin.Get(vcs.Type, vcsplugin.ProviderConfig{}).DeleteWebhook(
		ctx,
		&common.OauthContext{
			AccessToken: vcs.AccessToken,
		},
		vcs.InstanceURL,
		vcsConnector.Payload.ExternalId,
		vcsConnector.Payload.ExternalWebhookId,
	); err != nil {
		slog.Error("failed to delete webhook for VCS connector", slog.String("project", projectID), slog.String("VCS connector", vcsConnector.ResourceID), log.BBError(err))
	}

	return &emptypb.Empty{}, nil
}

func (s *VCSConnectorService) convertStoreVCSConnector(ctx context.Context, vcsConnector *store.VCSConnectorMessage) (*v1pb.VCSConnector, error) {
	creator, err := s.store.GetUserByID(ctx, vcsConnector.CreatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get creator: %v", err))
	}
	if creator == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("cannot find the creator: %d", vcsConnector.CreatorID))
	}
	updater, err := s.store.GetUserByID(ctx, vcsConnector.UpdaterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get updater: %v", err))
	}
	if updater == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("cannot find the updater: %d", vcsConnector.UpdaterID))
	}

	v1VCSConnector := &v1pb.VCSConnector{
		Name:          fmt.Sprintf("%s%s/%s%s", common.ProjectNamePrefix, vcsConnector.ProjectID, common.VCSConnectorPrefix, vcsConnector.ResourceID),
		CreateTime:    timestamppb.New(vcsConnector.CreatedTime),
		UpdateTime:    timestamppb.New(vcsConnector.UpdatedTime),
		Creator:       fmt.Sprintf("users/%s", creator.Email),
		Updater:       fmt.Sprintf("users/%s", updater.Email),
		Title:         vcsConnector.Payload.Title,
		VcsProvider:   fmt.Sprintf("%s%s", common.VCSProviderPrefix, vcsConnector.VCSResourceID),
		ExternalId:    vcsConnector.Payload.ExternalId,
		BaseDirectory: vcsConnector.Payload.BaseDirectory,
		Branch:        vcsConnector.Payload.Branch,
		FullPath:      vcsConnector.Payload.FullPath,
	}
	return v1VCSConnector, nil
}
