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

	return convertVCSProvider(vcs), nil
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
	vcsProvider, err := s.store.GetVCSProvider(ctx, &store.FindVCSProviderMessage{ResourceID: &request.VcsProviderId})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if vcsProvider != nil {
		return nil, status.Errorf(codes.AlreadyExists, "vcs provider %s already exists", request.VcsProviderId)
	}

	if !isValidResourceID(request.VcsProviderId) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vcs provider ID %v", request.VcsProviderId)
	}
	vcsProvider, err = convertV1VCSProvider(request)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	storeVCSProvider, err := s.store.CreateVCSProvider(ctx, principalID, vcsProvider)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create vcs provider: %v", err)
	}
	return convertVCSProvider(storeVCSProvider), nil
}

// UpdateVCSProvider updates an existing vcs provider.
func (s *VCSProviderService) UpdateVCSProvider(ctx context.Context, request *v1pb.UpdateVCSProviderRequest) (*v1pb.VCSProvider, error) {
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	vcsProvider, err := s.getVCS(ctx, request.VcsProvider.Name)
	if err != nil {
		return nil, err
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
	vcsProvider, err = s.store.UpdateVCSProvider(ctx, principalID, vcsProvider.ID, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertVCSProvider(vcsProvider), nil
}

// DeleteVCSProvider deletes an existing vcs provider.
func (s *VCSProviderService) DeleteVCSProvider(ctx context.Context, request *v1pb.DeleteVCSProviderRequest) (*emptypb.Empty, error) {
	vcsProvider, err := s.getVCS(ctx, request.Name)
	if err != nil {
		return nil, err
	}

	if err := s.store.DeleteVCSProvider(ctx, vcsProvider.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete vcs provider: %v", err)
	}
	return &emptypb.Empty{}, nil
}

// SearchVCSProviderRepositories searches vcs provider repositories, for example, GitHub repository.
func (s *VCSProviderService) SearchVCSProviderRepositories(ctx context.Context, request *v1pb.SearchVCSProviderRepositoriesRequest) (*v1pb.SearchVCSProviderRepositoriesResponse, error) {
	vcsProvider, err := s.getVCS(ctx, request.Name)
	if err != nil {
		return nil, err
	}

	apiExternalProjectList, err := vcs.Get(
		vcsProvider.Type,
		vcs.ProviderConfig{InstanceURL: vcsProvider.InstanceURL, AuthToken: vcsProvider.AccessToken},
	).FetchAllRepositoryList(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch external project list: %v", err)
	}

	var repositories []*v1pb.VCSRepository
	for _, apiExternalProject := range apiExternalProjectList {
		repositories = append(repositories, &v1pb.VCSRepository{
			Id:       apiExternalProject.ID,
			Title:    apiExternalProject.Name,
			FullPath: apiExternalProject.FullPath,
			WebUrl:   apiExternalProject.WebURL,
		})
	}

	return &v1pb.SearchVCSProviderRepositoriesResponse{
		Repositories: repositories,
	}, nil
}

// ListVCSConnectorsInProvider lists GitOps connectors for the provider.
func (s *VCSProviderService) ListVCSConnectorsInProvider(ctx context.Context, request *v1pb.ListVCSConnectorsInProviderRequest) (*v1pb.ListVCSConnectorsInProviderResponse, error) {
	vcs, err := s.getVCS(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	vcsConnectors, err := s.store.ListVCSConnectors(ctx, &store.FindVCSConnectorMessage{
		VCSUID: &vcs.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch external repository list: %v", err)
	}

	resp := &v1pb.ListVCSConnectorsInProviderResponse{}
	for _, vcsConnector := range vcsConnectors {
		v1VCSConnector, err := convertStoreVCSConnector(ctx, s.store, vcsConnector)
		if err != nil {
			return nil, err
		}
		resp.VcsConnectors = append(resp.VcsConnectors, v1VCSConnector)
	}
	return resp, nil
}

func (s *VCSProviderService) getVCS(ctx context.Context, name string) (*store.VCSProviderMessage, error) {
	vcsResourceID, err := common.GetVCSProviderID(name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	vcsProvider, err := s.store.GetVCSProvider(ctx, &store.FindVCSProviderMessage{ResourceID: &vcsResourceID})
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
		res = append(res, convertVCSProvider(vcsProvider))
	}
	return res
}

func convertVCSProvider(vcsProvider *store.VCSProviderMessage) *v1pb.VCSProvider {
	tp := v1pb.VCSType_VCS_TYPE_UNSPECIFIED
	switch vcsProvider.Type {
	case vcs.GitHub:
		tp = v1pb.VCSType_GITHUB
	case vcs.GitLab:
		tp = v1pb.VCSType_GITLAB
	case vcs.Bitbucket:
		tp = v1pb.VCSType_BITBUCKET
	case vcs.AzureDevOps:
		tp = v1pb.VCSType_AZURE_DEVOPS
	}

	return &v1pb.VCSProvider{
		Name:  fmt.Sprintf("%s%s", common.VCSProviderPrefix, vcsProvider.ResourceID),
		Title: vcsProvider.Title,
		Type:  tp,
		Url:   vcsProvider.InstanceURL,
	}
}

func convertV1VCSProvider(request *v1pb.CreateVCSProviderRequest) (*store.VCSProviderMessage, error) {
	v1VCSProvider := request.GetVcsProvider()
	if v1VCSProvider.GetTitle() == "" {
		return nil, errors.Errorf("Empty VCSProvider.Title")
	}
	if v1VCSProvider.GetType() == v1pb.VCSType_VCS_TYPE_UNSPECIFIED {
		return nil, errors.Errorf("Empty VCSProvider.Type")
	}
	if v1VCSProvider.GetUrl() == "" {
		return nil, errors.Errorf("Empty VCSProvider.Url")
	}
	if v1VCSProvider.GetAccessToken() == "" {
		return nil, errors.Errorf("Empty VCSProvider.Secret")
	}
	tp, err := convertVCSProviderTypeToVCSType(v1VCSProvider.GetType())
	if err != nil {
		return nil, err
	}

	storeVCSProvider := &store.VCSProviderMessage{
		ResourceID:  request.GetVcsProviderId(),
		Title:       v1VCSProvider.GetTitle(),
		Type:        tp,
		InstanceURL: strings.TrimRight(v1VCSProvider.GetUrl(), "/"),
		AccessToken: v1VCSProvider.GetAccessToken(),
	}

	return storeVCSProvider, nil
}

func convertVCSProviderTypeToVCSType(tp v1pb.VCSType) (vcs.Type, error) {
	switch tp {
	case v1pb.VCSType_GITHUB:
		return vcs.GitHub, nil
	case v1pb.VCSType_GITLAB:
		return vcs.GitLab, nil
	case v1pb.VCSType_BITBUCKET:
		return vcs.Bitbucket, nil
	case v1pb.VCSType_AZURE_DEVOPS:
		return vcs.AzureDevOps, nil
	}
	return "", errors.Errorf("unknown vcs provider type: %v", tp)
}
