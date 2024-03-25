package v1

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// VCSProviderService represents a service for managing vcs provider.
type VCSProviderService struct {
	v1pb.UnimplementedVCSProviderServiceServer
	store *store.Store
}

// NewVCSProviderService returns a new instance of VCSProviderService.
func NewVCSProviderService(store *store.Store) *VCSProviderService {
	return &VCSProviderService{store: store}
}

// GetVCSProvider get a single vcs provider.
func (s *VCSProviderService) GetVCSProvider(ctx context.Context, request *v1pb.GetVCSProviderRequest) (*v1pb.VCSProvider, error) {
	vcs, err := s.getVCS(ctx, request.Name)
	if err != nil {
		return nil, err
	}

	return convertToVCSProvider(vcs), nil
}

// ListVCSProviders lists vcs providers.
func (s *VCSProviderService) ListVCSProviders(ctx context.Context, _ *v1pb.ListVCSProvidersRequest) (*v1pb.ListVCSProvidersResponse, error) {
	vcsProviders, err := s.store.ListVCSProviders(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve vcs provider: %v", err)
	}
	return &v1pb.ListVCSProvidersResponse{VcsProviders: convertToVCSProviders(vcsProviders)}, nil
}

// CreateVCSProvider creates a new vcs provider.
func (s *VCSProviderService) CreateVCSProvider(ctx context.Context, request *v1pb.CreateVCSProviderRequest) (*v1pb.VCSProvider, error) {
	vcsProvider, err := checkAndConvertToStoreVersionControl(request.VcsProvider)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	storeVCSProvider, err := s.store.CreateVCSProviderV2(ctx, principalID, vcsProvider)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create vcs provider: %v", err)
	}
	return convertToVCSProvider(storeVCSProvider), nil
}

// UpdateVCSProvider updates an existing vcs provider.
func (s *VCSProviderService) UpdateVCSProvider(ctx context.Context, request *v1pb.UpdateVCSProviderRequest) (*v1pb.VCSProvider, error) {
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	vcsProviderUID, err := common.GetVCSProviderID(request.VcsProvider.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	vcsProvider, err := s.store.GetVCSProviderV2(ctx, vcsProviderUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve vcs provider: %v", err)
	}
	if vcsProvider == nil {
		return nil, status.Errorf(codes.NotFound, "vcs provider not found: %v", err)
	}

	update := &store.UpdateVCSProviderMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			if request.VcsProvider.Title == "" {
				return nil, status.Errorf(codes.InvalidArgument, "title should not be empty")
			}
			update.Name = &request.VcsProvider.Title
		case "access_token":
			if request.VcsProvider.AccessToken == "" {
				return nil, status.Errorf(codes.InvalidArgument, "secret should not be empty")
			}
			update.AccessToken = &request.VcsProvider.AccessToken
		}
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	vcsProvider, err = s.store.UpdateVCSProviderV2(ctx, principalID, vcsProviderUID, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToVCSProvider(vcsProvider), nil
}

// DeleteVCSProvider deletes an existing vcs provider.
func (s *VCSProviderService) DeleteVCSProvider(ctx context.Context, request *v1pb.DeleteVCSProviderRequest) (*emptypb.Empty, error) {
	vcsProviderUID, err := common.GetVCSProviderID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	vcsProvider, err := s.store.GetVCSProviderV2(ctx, vcsProviderUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve vcs provider: %v", err)
	}
	if vcsProvider == nil {
		return nil, status.Errorf(codes.NotFound, "vcs provider not found: %v", err)
	}

	if err := s.store.DeleteVCSProviderV2(ctx, vcsProviderUID); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete vcs provider: %v", err)
	}
	return &emptypb.Empty{}, nil
}

// SearchVCSProviderProjects searches vcs provider projects, for example, GitHub repository.
func (s *VCSProviderService) SearchVCSProviderProjects(ctx context.Context, request *v1pb.SearchVCSProviderProjectsRequest) (*v1pb.SearchVCSProviderProjectsResponse, error) {
	vcsProviderUID, err := common.GetVCSProviderID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	vcsProvider, err := s.store.GetVCSProviderV2(ctx, vcsProviderUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve vcs provider: %v", err)
	}
	if vcsProvider == nil {
		return nil, status.Errorf(codes.NotFound, "vcs provider not found: %v", err)
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace setting with error: %v", err.Error())
	}
	if setting.ExternalUrl == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "external url is required")
	}

	apiExternalProjectList, err := vcs.Get(vcsProvider.Type, vcs.ProviderConfig{}).FetchAllRepositoryList(
		ctx,
		&common.OauthContext{
			AccessToken: vcsProvider.AccessToken,
		},
		vcsProvider.InstanceURL,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch external project list: %v", err)
	}

	var externalProjects []*v1pb.SearchVCSProviderProjectsResponse_Project
	for _, apiExternalProject := range apiExternalProjectList {
		externalProjects = append(externalProjects, &v1pb.SearchVCSProviderProjectsResponse_Project{
			Id:       apiExternalProject.ID,
			Title:    apiExternalProject.Name,
			Fullpath: apiExternalProject.FullPath,
			WebUrl:   apiExternalProject.WebURL,
		})
	}

	return &v1pb.SearchVCSProviderProjectsResponse{
		Projects: externalProjects,
	}, nil
}

// ListProjectGitOpsInfo lists GitOps info of a project.
func (s *VCSProviderService) ListProjectGitOpsInfo(ctx context.Context, request *v1pb.ListProjectGitOpsInfoRequest) (*v1pb.ListProjectGitOpsInfoResponse, error) {
	vcsProviderUID, err := common.GetVCSProviderID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	repoList, err := s.store.ListRepositoryV2(ctx, &store.FindRepositoryMessage{
		VCSUID: &vcsProviderUID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch external repository list: %v", err)
	}

	resp := &v1pb.ListProjectGitOpsInfoResponse{}
	for _, repo := range repoList {
		resp.ProjectGitopsInfo = append(resp.ProjectGitopsInfo, convertToProjectGitOpsInfo(repo))
	}

	return resp, nil
}

func (s *VCSProviderService) getVCS(ctx context.Context, name string) (*store.VCSProviderMessage, error) {
	vcsProviderUID, err := common.GetVCSProviderID(name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	vcsProvider, err := s.store.GetVCSProviderV2(ctx, vcsProviderUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve vcs provider: %v", err)
	}
	if vcsProvider == nil {
		return nil, status.Errorf(codes.NotFound, "vcs provider not found: %v", err)
	}

	return vcsProvider, nil
}

func convertToVCSProviders(vcsProviders []*store.VCSProviderMessage) []*v1pb.VCSProvider {
	var res []*v1pb.VCSProvider
	for _, vcsProvider := range vcsProviders {
		res = append(res, convertToVCSProvider(vcsProvider))
	}
	return res
}

func convertToVCSProvider(vcsProvider *store.VCSProviderMessage) *v1pb.VCSProvider {
	tp := v1pb.VCSProvider_TYPE_UNSPECIFIED
	switch vcsProvider.Type {
	case vcs.GitHub:
		tp = v1pb.VCSProvider_GITHUB
	case vcs.GitLab:
		tp = v1pb.VCSProvider_GITLAB
	case vcs.Bitbucket:
		tp = v1pb.VCSProvider_BITBUCKET
	case vcs.AzureDevOps:
		tp = v1pb.VCSProvider_AZURE_DEVOPS
	}

	return &v1pb.VCSProvider{
		Name:  fmt.Sprintf("%s%d", common.VCSProviderPrefix, vcsProvider.ID),
		Title: vcsProvider.Name,
		Type:  tp,
		Url:   vcsProvider.InstanceURL,
	}
}

func checkAndConvertToStoreVersionControl(vcsProvider *v1pb.VCSProvider) (*store.VCSProviderMessage, error) {
	if vcsProvider.Title == "" {
		return nil, errors.Errorf("Empty VCSProvider.Title")
	}
	if vcsProvider.Type == v1pb.VCSProvider_TYPE_UNSPECIFIED {
		return nil, errors.Errorf("Empty VCSProvider.Type")
	}
	if vcsProvider.Url == "" {
		return nil, errors.Errorf("Empty VCSProvider.Url")
	}
	if vcsProvider.AccessToken == "" {
		return nil, errors.Errorf("Empty VCSProvider.Secret")
	}
	storeVCSProvider := &store.VCSProviderMessage{
		Name:        vcsProvider.Title,
		AccessToken: vcsProvider.AccessToken,
	}

	tp, err := convertVCSProviderTypeToVCSType(vcsProvider.Type)
	if err != nil {
		return nil, err
	}

	storeVCSProvider.InstanceURL = strings.TrimRight(vcsProvider.Url, "/")
	storeVCSProvider.Type = tp
	return storeVCSProvider, nil
}

func convertVCSProviderTypeToVCSType(tp v1pb.VCSProvider_Type) (vcs.Type, error) {
	switch tp {
	case v1pb.VCSProvider_GITHUB:
		return vcs.GitHub, nil
	case v1pb.VCSProvider_GITLAB:
		return vcs.GitLab, nil
	case v1pb.VCSProvider_BITBUCKET:
		return vcs.Bitbucket, nil
	case v1pb.VCSProvider_AZURE_DEVOPS:
		return vcs.AzureDevOps, nil
	}
	return "", errors.Errorf("unknown vcs provider type: %v", tp)
}
