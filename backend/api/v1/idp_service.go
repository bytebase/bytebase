package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// IdentityProviderService implements the identity provider service.
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
	if err := validIdentityProviderConfig(request.IdentityProvider.Type, request.IdentityProvider.Config); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	identityProviderMessage := store.IdentityProviderMessage{
		ResourceID: request.IdentityProviderId,
		Title:      request.IdentityProvider.Title,
		Domain:     request.IdentityProvider.Domain,
		Type:       convertIdentityProviderTypeToStore(request.IdentityProvider.Type),
		Config:     convertIdentityProviderConfigToStore(request.IdentityProvider.GetConfig()),
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
		case "identity_provider.title":
			patch.Title = &request.IdentityProvider.Title
		case "identity_provider.domain":
			patch.Domain = &request.IdentityProvider.Domain
		case "identity_provider.config":
			patch.Config = convertIdentityProviderConfigToStore(request.IdentityProvider.Config)
		}
	}
	if patch.Config != nil {
		if err := validIdentityProviderConfig(convertIdentityProviderTypeFromStore(identityProvider.Type), request.IdentityProvider.Config); err != nil {
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
	identityProviderType := convertIdentityProviderTypeFromStore(identityProvider.Type)
	config := convertIdentityProviderConfigFromStore(identityProvider.Config)
	return &v1pb.IdentityProvider{
		Name:   fmt.Sprintf("%s%s", identityProviderNamePrefix, identityProvider.ResourceID),
		Uid:    fmt.Sprintf("%d", identityProvider.UID),
		State:  convertDeletedToState(identityProvider.Deleted),
		Title:  identityProvider.Title,
		Domain: identityProvider.Domain,
		Type:   identityProviderType,
		Config: config,
	}
}

func convertIdentityProviderConfigFromStore(identityProviderConfig *storepb.IdentityProviderConfig) *v1pb.IdentityProviderConfig {
	if v := identityProviderConfig.GetOauth2Config(); v != nil {
		fieldMapping := v1pb.FieldMapping{
			Identifier:  v.FieldMapping.Identifier,
			DisplayName: v.FieldMapping.DisplayName,
			Email:       v.FieldMapping.Email,
		}
		return &v1pb.IdentityProviderConfig{
			Config: &v1pb.IdentityProviderConfig_Oauth2Config{
				Oauth2Config: &v1pb.OAuth2IdentityProviderConfig{
					AuthUrl:      v.AuthUrl,
					TokenUrl:     v.TokenUrl,
					UserInfoUrl:  v.UserInfoUrl,
					ClientId:     v.ClientId,
					ClientSecret: v.ClientSecret,
					Scopes:       v.Scopes,
					FieldMapping: &fieldMapping,
				},
			},
		}
	} else if v := identityProviderConfig.GetOidcConfig(); v != nil {
		fieldMapping := v1pb.FieldMapping{
			Identifier:  v.FieldMapping.Identifier,
			DisplayName: v.FieldMapping.DisplayName,
			Email:       v.FieldMapping.Email,
		}
		return &v1pb.IdentityProviderConfig{
			Config: &v1pb.IdentityProviderConfig_OidcConfig{
				OidcConfig: &v1pb.OIDCIdentityProviderConfig{
					Issuer:       v.Issuer,
					ClientId:     v.ClientId,
					ClientSecret: v.ClientSecret,
					FieldMapping: &fieldMapping,
				},
			},
		}
	} else {
		return nil
	}
}

func convertIdentityProviderConfigToStore(identityProviderConfig *v1pb.IdentityProviderConfig) *storepb.IdentityProviderConfig {
	if v := identityProviderConfig.GetOauth2Config(); v != nil {
		fieldMapping := storepb.FieldMapping{
			Identifier:  v.FieldMapping.Identifier,
			DisplayName: v.FieldMapping.DisplayName,
			Email:       v.FieldMapping.Email,
		}
		return &storepb.IdentityProviderConfig{
			Config: &storepb.IdentityProviderConfig_Oauth2Config{
				Oauth2Config: &storepb.OAuth2IdentityProviderConfig{
					AuthUrl:      v.AuthUrl,
					TokenUrl:     v.TokenUrl,
					UserInfoUrl:  v.UserInfoUrl,
					ClientId:     v.ClientId,
					ClientSecret: v.ClientSecret,
					Scopes:       v.Scopes,
					FieldMapping: &fieldMapping,
				},
			},
		}
	} else if v := identityProviderConfig.GetOidcConfig(); v != nil {
		fieldMapping := storepb.FieldMapping{
			Identifier:  v.FieldMapping.Identifier,
			DisplayName: v.FieldMapping.DisplayName,
			Email:       v.FieldMapping.Email,
		}
		return &storepb.IdentityProviderConfig{
			Config: &storepb.IdentityProviderConfig_OidcConfig{
				OidcConfig: &storepb.OIDCIdentityProviderConfig{
					Issuer:       v.Issuer,
					ClientId:     v.ClientId,
					ClientSecret: v.ClientSecret,
					FieldMapping: &fieldMapping,
				},
			},
		}
	} else {
		return nil
	}
}

func convertIdentityProviderTypeFromStore(identityProviderType storepb.IdentityProviderType) v1pb.IdentityProviderType {
	if identityProviderType == storepb.IdentityProviderType_OAUTH2 {
		return v1pb.IdentityProviderType_OAUTH2
	} else if identityProviderType == storepb.IdentityProviderType_OIDC {
		return v1pb.IdentityProviderType_OIDC
	}
	return v1pb.IdentityProviderType_IDENTITY_PROVIDER_TYPE_UNSPECIFIED
}

func convertIdentityProviderTypeToStore(identityProviderType v1pb.IdentityProviderType) storepb.IdentityProviderType {
	if identityProviderType == v1pb.IdentityProviderType_OAUTH2 {
		return storepb.IdentityProviderType_OAUTH2
	} else if identityProviderType == v1pb.IdentityProviderType_OIDC {
		return storepb.IdentityProviderType_OIDC
	}
	return storepb.IdentityProviderType_IDENTITY_PROVIDER_TYPE_UNSPECIFIED
}

// validIdentityProviderConfig validates the identity provider's config is a valid JSON.
func validIdentityProviderConfig(identityProviderType v1pb.IdentityProviderType, identityProviderConfig *v1pb.IdentityProviderConfig) error {
	if identityProviderType == v1pb.IdentityProviderType_OAUTH2 {
		if identityProviderConfig.GetOauth2Config() == nil {
			return errors.Errorf("unexpected provider config value")
		}
	} else if identityProviderType == v1pb.IdentityProviderType_OIDC {
		if identityProviderConfig.GetOidcConfig() == nil {
			return errors.Errorf("unexpected provider config value")
		}
	} else {
		return errors.Errorf("unexpected provider type %s", identityProviderType)
	}
	return nil
}
