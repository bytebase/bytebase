package v1

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

type IdentityProviderService struct {
	v1pb.UnimplementedIdentityProviderServiceServer
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
}

// NewIdentityProviderService creates a new IdentityProviderService.
func NewIdentityProviderService(store *store.Store, licenseService enterpriseAPI.LicenseService) *IdentityProviderService {
	return &IdentityProviderService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetIdentityProvider gets an identity provider.
func (s *IdentityProviderService) GetIdentityProvider(ctx context.Context, request *v1pb.GetIdentityProviderRequest) (*v1pb.IdentityProvider, error) {
	identityProvider, err := s.getIdentityProviderMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	return convertToIdentityProvider(identityProvider), nil
}

// ListIdentityProviders lists all identity providers.
func (s *IdentityProviderService) ListIdentityProviders(ctx context.Context, request *v1pb.ListIdentityProvidersRequest) (*v1pb.ListIdentityProvidersResponse, error) {
	identityProviders, err := s.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{ShowDeleted: request.ShowDeleted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListIdentityProvidersResponse{}
	for _, identityProvider := range identityProviders {
		response.IdentityProviders = append(response.IdentityProviders, convertToIdentityProvider(identityProvider))
	}
	return response, nil
}

// CreateIdentityProvider creates an identity provider.
func (s *IdentityProviderService) CreateIdentityProvider(ctx context.Context, request *v1pb.CreateIdentityProviderRequest) (*v1pb.IdentityProvider, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if request.IdentityProvider == nil {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider must be set")
	}

	if !isValidResourceID(request.IdentityProviderId) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid identity provider ID %v", request.IdentityProviderId)
	}
	identityProviderMessage := store.IdentityProviderMessage{
		ResourceID: request.IdentityProviderId,
		Title:      request.IdentityProvider.Title,
		Domain:     request.IdentityProvider.Domain,
		Type:       api.IdentityProviderType(request.IdentityProvider.Type.String()),
		Config:     request.IdentityProvider.Config,
	}
	if err := validIdentityProviderConfig(identityProviderMessage.Type, identityProviderMessage.Config); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	identityProvider, err := s.store.CreateIdentityProvider(ctx, &identityProviderMessage, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToIdentityProvider(identityProvider), nil
}

// UpdateIdentityProvider updates an identity provider.
func (s *IdentityProviderService) UpdateIdentityProvider(ctx context.Context, request *v1pb.UpdateIdentityProviderRequest) (*v1pb.IdentityProvider, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if request.IdentityProvider == nil {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	identityProvider, err := s.getIdentityProviderMessage(ctx, request.IdentityProvider.Name)
	if err != nil {
		return nil, err
	}
	if identityProvider.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider %q has been deleted", request.IdentityProvider.Name)
	}

	patch := &store.UpdateIdentityProviderMessage{
		UpdaterID:  principalID,
		ResourceID: identityProvider.ResourceID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "identityProvider.title":
			patch.Title = &request.IdentityProvider.Title
		case "identityProvider.domain":
			patch.Domain = &request.IdentityProvider.Domain
		case "identityProvider.config":
			patch.Config = &request.IdentityProvider.Config
		}
	}
	if patch.Config != nil {
		if err := validIdentityProviderConfig(identityProvider.Type, *patch.Config); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
	}

	identityProvider, err = s.store.UpdateIdentityProvider(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToIdentityProvider(identityProvider), nil
}

// DeleteIdentityProvider deletes an identity provider.
func (s *IdentityProviderService) DeleteIdentityProvider(ctx context.Context, request *v1pb.DeleteIdentityProviderRequest) (*emptypb.Empty, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)

	identityProvider, err := s.getIdentityProviderMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if identityProvider.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider %q has been deleted", request.Name)
	}

	patch := &store.UpdateIdentityProviderMessage{
		UpdaterID:  principalID,
		ResourceID: identityProvider.ResourceID,
		Delete:     &deletePatch,
	}
	if _, err := s.store.UpdateIdentityProvider(ctx, patch); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// UndeleteIdentityProvider undeletes an identity provider.
func (s *IdentityProviderService) UndeleteIdentityProvider(ctx context.Context, request *v1pb.UndeleteIdentityProviderRequest) (*v1pb.IdentityProvider, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)

	identityProvider, err := s.getIdentityProviderMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if !identityProvider.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider %q is active", request.Name)
	}

	patch := &store.UpdateIdentityProviderMessage{
		UpdaterID:  principalID,
		ResourceID: identityProvider.ResourceID,
		Delete:     &deletePatch,
	}
	identityProvider, err = s.store.UpdateIdentityProvider(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToIdentityProvider(identityProvider), nil
}

func (s *IdentityProviderService) getIdentityProviderMessage(ctx context.Context, name string) (*store.IdentityProviderMessage, error) {
	identityProviderID, err := getIdentityProviderID(name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	identityProvider, err := s.store.GetIdentityProvider(ctx, &store.FindIdentityProviderMessage{
		ResourceID: &identityProviderID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if identityProvider == nil {
		return nil, status.Errorf(codes.NotFound, "identity provider %q not found", name)
	}

	return identityProvider, nil
}

func convertToIdentityProvider(identityProvider *store.IdentityProviderMessage) *v1pb.IdentityProvider {
	identityProviderType := v1pb.IdentityProviderType_IDENTITY_PROVIDER_UNSPECIFIED
	if identityProvider.Type == api.OAuth2IdentityProvider {
		identityProviderType = v1pb.IdentityProviderType_OAUTH2
	} else if identityProvider.Type == api.OIDCIdentityProvider {
		identityProviderType = v1pb.IdentityProviderType_OIDC
	}

	return &v1pb.IdentityProvider{
		Name:   fmt.Sprintf("%s%s", identityProviderNamePrefix, identityProvider.ResourceID),
		Uid:    fmt.Sprintf("%d", identityProvider.UID),
		State:  convertDeletedToState(identityProvider.Deleted),
		Title:  identityProvider.Title,
		Domain: identityProvider.Domain,
		Type:   identityProviderType,
		Config: identityProvider.Config,
	}
}

// validIdentityProviderConfig validates the identity provider's config is a valid JSON.
func validIdentityProviderConfig(identityProviderType api.IdentityProviderType, configString string) error {
	if identityProviderType == api.OAuth2IdentityProvider {
		formatedConfig := &storepb.OAuth2IdentityProviderConfig{}
		if err := json.Unmarshal([]byte(configString), formatedConfig); err != nil {
			return errors.Wrap(err, "failed to unmarshal config")
		}
	} else if identityProviderType == api.OIDCIdentityProvider {
		formatedConfig := &storepb.OIDCIdentityProviderConfig{}
		if err := json.Unmarshal([]byte(configString), formatedConfig); err != nil {
			return errors.Wrap(err, "failed to unmarshal config")
		}
	} else {
		return errors.Errorf("unexpected provider type %s", identityProviderType)
	}
	return nil
}
